package main

import (
	"math"
	"math/rand"
)

const REFACTORY_PERIOD = 5.0

const ACTIVATION_DECAY_RATE = 0.98

// Homeostasis: each neuron tracks its own firing rate and nudges its threshold to keep that rate near a target
const TARGET_FIRING_RATE = 0.005       // Spikes per cycle (1 spike every 200 cycles)
const FIRING_RATE_WINDOW = 1000.0      // Time constant of the firing rate average, in cycles
const HOMEOSTASIS_LEARNING_RATE = 0.05 // At double the target rate, the offset rises ~0.25 per 1000 cycles
const HOMEOSTASIS_OFFSET_MIN = -0.5
const HOMEOSTASIS_OFFSET_MAX = 3.0

// Neighbour synchrony: when a neuron fires, it averages the time since its 8 surrounding neurons fired
// The closer that average is to 0, the more its contributing inputs are punished, to force neighbours to differentiate
const NEIGHBOUR_FIRE_WINDOW = 5.0 // Time constant in cycles, an average of this many cycles gives ~0.37 of the full punish
const NEIGHBOUR_SYNC_PUNISH = 0.1 // Fraction of strength removed when the average time since neighbours fired is 0

// Quiet neuron bonus: the longer a neuron has gone without firing, the more its contributing inputs are rewarded when it finally fires
// Saturates towards 1 + QUIET_REWARD_MAX_BONUS, so a long silent neuron can be recruited back into the network
//
//	200 cycles   -> x1.18
//	1000 cycles  -> x1.79
//	3000 cycles  -> x2.55
//	10000+ cycles -> ~x3.0
const QUIET_REWARD_MAX_BONUS = 3.0   // Extra reward multiplier reached after a very long silence
const QUIET_REWARD_TIMESCALE = 500.0 // Cycles of silence to reach ~63% of the max bonus

// When there is a presynaptic fire then a postsynaptic fire with meaningful contribution, then it will reward the connection endrite this amount
const POST_SYNAPTIC_FIRE_REWARD = 0.1

// Neuron represents a single neuron in the network.
type Neuron struct {
	Brain        *Brain
	ParentSquare *BrainSquare
	Incoming     []*Connection
	Outgoing     []*Connection

	// Basic state (can be expanded)
	LastfireTime              float64
	AlreadyAddedToTocheckList bool
	IsExcitatory              bool

	Threshold float64

	RefactoryPrediod float64

	Activation     float64
	LastUpdateTime float64

	FiringRate        float64 // Running average of spikes per cycle
	HomeostaticOffset float64 // Added to the threshold, rises if firing too much, falls if too little
}

// NewNeuron creates and returns a new neuron.
func NewNeuron(brain *Brain, parentSquare *BrainSquare, isExcitatory bool) Neuron {
	return Neuron{
		Brain:                     brain,
		ParentSquare:              parentSquare,
		Incoming:                  make([]*Connection, 0),
		Outgoing:                  make([]*Connection, 0),
		Activation:                rand.Float64()*0.5 + 0.5, //0.5 to 1
		LastfireTime:              0.0,
		LastUpdateTime:            0.0,
		Threshold:                 1.0 + ((rand.Float64() - 0.5) / 3), // Random threshold between 0.5 and 1.0
		RefactoryPrediod:          REFACTORY_PERIOD + rand.Float64()*3,
		IsExcitatory:              isExcitatory,
		AlreadyAddedToTocheckList: false,
		FiringRate:                TARGET_FIRING_RATE, // Start at target so there is no startup transient
		HomeostaticOffset:         0.0,
	}

}

// AddIncomingConnection adds a neuron that sends signals *to* this neuron.
func (n *Neuron) AddIncomingConnection(con *Connection) {
	n.Incoming = append(n.Incoming, con)
}

// AddOutgoingConnection adds a neuron that this neuron sends signals *to*.
func (n *Neuron) AddOutgoingConnection(con *Connection) {
	n.Outgoing = append(n.Outgoing, con)
}

func (n *Neuron) InitialConnect(brain *Brain, brainSquare *BrainSquare) {

	if n.IsExcitatory {
		connectExcitatory(brain, brainSquare, n)
	} else {
		connectInhibitory(brain, n)
	}

	// Shuffle outgoing
	rand.Shuffle(len(n.Outgoing), func(i, j int) {
		n.Outgoing[i], n.Outgoing[j] = n.Outgoing[j], n.Outgoing[i]
	})
}

func (n *Neuron) CheckForFiring(time float64) {

	n.UpdateActivationAndWeightOnly(time)

	// Recalculate threshold
	var lastFireTimeDelta = time - n.LastfireTime
	var desireToFireComp = -math.Min(lastFireTimeDelta/8000, 0.8)
	//var absoluteRefactoryPeriod = REFACTORY_PERIOD * 2
	//var justFiredDepolarization = math.Pow(math.Max(absoluteRefactoryPeriod-lastFireTimeDelta, 0)/absoluteRefactoryPeriod, 0.5) * 3 // Multiply the SPIKE_DURATION by 2 to get teh absolute refactory period
	virtualThreshold := n.Threshold + desireToFireComp + n.HomeostaticOffset // + justFiredDepolarization

	if n.Activation >= virtualThreshold && lastFireTimeDelta > n.RefactoryPrediod { // SUCCESSFUL FIRE!!!

		n.Threshold = Lerp(n.Threshold, 1.1, 0.3)

		// Because the simulation is in timesteps, it can be overshot, whereas in nature, as soon at the spike occurs, it will weaken any more

		//var overShootThreshold = 1.5 // Need to add 0.5 because a single normal fire may overshoot naturally
		var overshoot = math.Max(n.Activation-1.2, 0.0)
		hasOvershot := overshoot > 0
		//var overshootCorrector = math.Tanh(overshoot/2.0) * 0.3

		overShootWeightsTotal := 0.0
		if hasOvershot {
			for _, c := range n.Incoming {

				if c.IsAwaitingReward {
					var conFiredDurationAgo = time - c.From.LastfireTime

					if conFiredDurationAgo <= 1.0001 {
						if n.IsExcitatory {
							//c.Strength *= 1.0 - (overshoot / 5.0)
							overShootWeightsTotal += c.Strength
						}
					}
				}
			}
		}

		// Punish scales down exponentially with the neighbours' average time since firing
		// Neighbours that fired closer in time give a higher punish, further away gives a lower punish
		//   avg 0 cycles  -> 0.3    (full NEIGHBOUR_SYNC_PUNISH)
		//   avg 1 cycle   -> ~0.21  (72% of full)
		//   avg 3 cycles  -> ~0.11  (37% of full)
		//   avg 6 cycles  -> ~0.041 (14% of full)
		//   avg 10 cycles -> ~0.011 (4% of full, effectively none)
		neighbourSyncPunish := math.Pow(n.AverageNeighbourTimeSinceFired(time)/NEIGHBOUR_FIRE_WINDOW, 1.8) * NEIGHBOUR_SYNC_PUNISH

		// Reward multiplier for how long this neuron was silent, LastfireTime hasn't been reset yet so lastFireTimeDelta is the silence
		quietRewardBonus := 1.0 + QUIET_REWARD_MAX_BONUS*(1.0-math.Exp(-lastFireTimeDelta/QUIET_REWARD_TIMESCALE))

		// Send reward or punsih to presynaptic dendrites
		for _, c := range n.Incoming {
			c.UpdateWeightsSince(time)
			var conFiredDurationAgo = time - c.From.LastfireTime

			if c.IsAwaitingReward {

				// Rewards both excitory and inhoibotory neurons
				contribution := math.Pow(ACTIVATION_DECAY_RATE, conFiredDurationAgo)

				idealStrength := contribution * quietRewardBonus
				delta := math.Abs(c.Strength - idealStrength)
				c.Strength = Lerp(c.Strength, idealStrength, POST_SYNAPTIC_FIRE_REWARD+POST_SYNAPTIC_FIRE_REWARD/delta)

				//c.Strength = RewardValue(c.Strength, 0.06*contribution*quietRewardBonus, 1.5)

				if hasOvershot && conFiredDurationAgo <= 1.0001 {
					if n.IsExcitatory {
						relativeConnectionRatio := c.Strength / overShootWeightsTotal
						c.Strength *= 1.0 - (relativeConnectionRatio * (overshoot / 2.0))
					}
				}

				// Punish inputs that drove this neuron to fire at the same time as its neighbours
				c.Strength = PunishValue(c.Strength, neighbourSyncPunish)

				c.IsAwaitingReward = false
			}
		}

		// Send signal to postsynaptic neurons
		n.LastfireTime = time
		n.Activation = 0
		n.FiringRate += 1.0 / FIRING_RATE_WINDOW
		for _, c := range n.Outgoing {

			c.UpdateWeightsSince(time)
			c.To.UpdateActivationAndWeightOnly(time)

			timeSinceLastFire := time - c.To.LastfireTime

			inRefatoryPeriod := timeSinceLastFire < c.To.RefactoryPrediod

			if c.IsExcitatory {

				if inRefatoryPeriod { // If it has fired into a already spiked cell, weaken

					rat := 1 - timeSinceLastFire/c.To.RefactoryPrediod

					c.Strength = PunishValue(c.Strength, 0.05*rat)

				} else { // Normal firing

					c.To.Activation += c.Strength // * PosOnlyRandMultiplier(c.Strength*0.001) // We could add a magnitude multiplyer to the cell, default to 0.15 to allow the growth to always be between 0 and 1
					c.IsAwaitingReward = true
				}

				// Because we have added activation to the next neuron we will check if it has fired
				if !c.To.AlreadyAddedToTocheckList {
					c.To.AlreadyAddedToTocheckList = true
					c.To.Brain.NextNeuronsToCheck = append(c.To.Brain.NextNeuronsToCheck, c.To)
				}

			} else { // negative neuron outgoing

				if !inRefatoryPeriod {
					c.To.Activation -= c.Strength
				}
			}

			c.Strength *= 0.999 //Tiny amount of punish to stop rapid firing
		}
	}

	/*else if n.Activation > n.Threshold*0.8 { // Nearly fired
		// This block is for NMDA
		// Go through the connections and see who recently fired, very slighly reward with more weight

		var rewardAmount = n.Activation - (n.Threshold * 0.8)

		// This effect is very minor, if it nearly fired but doesnt, reward the connection
		for _, c := range n.Incoming {
			var firedDurationAgo = time - c.From.LastfireTime
			if c.IsAwaitingReward && firedDurationAgo < CONTRIBUTION_REWARD_WINDOW/2 { // Half the of the contribution reward window
				var timingReward = 1.0 / math.Pow(firedDurationAgo+5, 1.5) //+5 to reduce the ratio between 1 and 2
				if c.IsExcitatory {
					c.Strength = RewardValue(c.Strength, 0.3*rewardAmount*timingReward, 1.5) // This needs to be belenced with the amount of upfront punish
				} else { // Want to punish negative neurons because they dont like activation, this effect should be even more mild because
					//c.Strength = PunishValue(c.Strength, 0.1*rewardAmount*timingReward) // This needs to be belenced with the amount of upfront punish
				}
			}
		}
	}*/
}

// AverageNeighbourTimeSinceFired averages the cycles since the neurons in the 8 surrounding squares last fired
func (n *Neuron) AverageNeighbourTimeSinceFired(time float64) float64 {

	var sourceX = n.ParentSquare.PosX
	var sourceY = n.ParentSquare.PosY

	total := 0.0
	count := 0

	for y := -1; y <= 1; y++ {
		for x := -1; x <= 1; x++ {

			if x == 0 && y == 0 {
				continue // Skip own square
			}

			wrappedX, wrappedY := n.Brain.WrapPosition(sourceX+x, sourceY+y)
			square := &n.Brain.BrainSquares[wrappedX][wrappedY]

			if square.isWall || len(square.Neurons) == 0 {
				continue
			}

			neighbour := &square.Neurons[0] // The excitatory neuron
			total += time - neighbour.LastfireTime
			count++
		}
	}

	if count == 0 {
		return math.Inf(1) // No neighbours, so no punish
	}

	return total / float64(count-2)
}

func (n *Neuron) UpdateActivationAndWeightOnly(time float64) {

	if time == n.LastUpdateTime {
		return
	}

	var cyclesSince = time - n.LastUpdateTime

	n.Activation *= math.Pow(ACTIVATION_DECAY_RATE, cyclesSince)

	n.Threshold *= math.Pow(0.99, cyclesSince)

	// Homeostasis, integrated exactly over the gap since the last update since neurons are only updated when touched
	// The firing rate decays exponentially, so its integral over the gap is rate * window * (1 - decay)
	var rateDecay = math.Exp(-cyclesSince / FIRING_RATE_WINDOW)
	var integratedRate = n.FiringRate * FIRING_RATE_WINDOW * (1.0 - rateDecay)
	var rateError = integratedRate - TARGET_FIRING_RATE*cyclesSince // Positive when firing too much
	n.HomeostaticOffset += HOMEOSTASIS_LEARNING_RATE * rateError
	n.HomeostaticOffset = math.Max(HOMEOSTASIS_OFFSET_MIN, math.Min(HOMEOSTASIS_OFFSET_MAX, n.HomeostaticOffset))
	n.FiringRate *= rateDecay

	n.LastUpdateTime = time
}

func (n *Neuron) AddActivation(amount float64, time float64) {
	n.UpdateActivationAndWeightOnly(time)
	n.Activation += amount
	if n.Activation < 0 {
		n.Activation = 0
	}
	n.CheckForFiring(time)
}

func connectExcitatory(brain *Brain, brainSquare *BrainSquare, n *Neuron) {

	var sourceX = n.ParentSquare.PosX
	var sourceY = n.ParentSquare.PosY

	var radiusI = 10
	var radiusF = float64(radiusI)

	var numberOutgoing = 0

	// Not sure why these are flipped?
	var circlePositionX = int(float64(sourceX) + float64(brainSquare.DirVecX*(radiusF+1.0)))
	var circlePositionY = int(float64(sourceY) + float64(brainSquare.DirVecY*(radiusF+1.0)))

	for y := -radiusI; y <= radiusI; y++ {
		for x := -radiusI; x <= radiusI; x++ {

			var dist = math.Sqrt(math.Pow(float64(x), 2) + math.Pow(float64(y), 2))

			if dist > radiusF || rand.Float32() > 0.4 {
				continue
			}

			mappedPosX := x + circlePositionX
			mappedPosY := y + circlePositionY

			if mappedPosX == sourceX && mappedPosY == sourceY {
				continue // Dont connect to self
			}

			// Distance from the source neuron, not from the circle centre, measured before wrapping
			var sourceDist = math.Hypot(float64(mappedPosX-sourceX), float64(mappedPosY-sourceY))

			wrappedX, wrappedY := brain.WrapPosition(mappedPosX, mappedPosY)

			targetNeuronsSlice := brain.BrainSquares[wrappedX][wrappedY].Neurons

			for i := 0; i < len(targetNeuronsSlice); i++ {
				neuronToConnectTo := &targetNeuronsSlice[i]

				var con = Connection{
					From:         n,
					To:           neuronToConnectTo,
					IsExcitatory: true,
				}
				con.SetDistanceBase(sourceDist)
				con.Strength = con.BaseStrength * (0.75 + rand.Float64()*0.5) // Start within 25% of the base

				n.AddOutgoingConnection(&con)
				neuronToConnectTo.AddIncomingConnection(&con)
				numberOutgoing += 1
			}
		}
	}
}

func connectInhibitory(brain *Brain, n *Neuron) {

	var sourceX = n.ParentSquare.PosX
	var sourceY = n.ParentSquare.PosY

	var spread = int(6)
	var depth = int(6)

	var spreadStart = sourceY - spread/2
	var spreadEnd = sourceY + spread/2

	var depthStart = sourceX - depth/2
	var depthEnd = sourceX + depth/2

	var numberOutgoing = 0

	for y := spreadStart; y <= spreadEnd; y++ {

		for x := depthStart; x <= depthEnd; x++ {

			wrappedX, wrappedY := brain.WrapPosition(x, y)

			if wrappedX == sourceX && wrappedY == sourceY {
				continue // Dont connect to self
			}

			targetNeuronsSlice := brain.BrainSquares[wrappedX][wrappedY].Neurons

			for i := 0; i < len(targetNeuronsSlice); i++ {
				neuronToConnectTo := &targetNeuronsSlice[i]

				var con = Connection{
					From:         n,
					To:           neuronToConnectTo,
					IsExcitatory: false,
					Strength:     rand.Float64() * 0.05,
					BaseStrength: 0.0, // Inhibitory connections still decay towards 0
					RelaxDecay:   0.999,
				}

				n.AddOutgoingConnection(&con)
				neuronToConnectTo.AddIncomingConnection(&con)

				numberOutgoing += 1
			}
		}
	}

	//fmt.Println("Number outgoing: ", numberOutgoing)
}
