package main

import (
	"math"
	"math/rand"
)

// These are vars rather than consts so the slider panel (ui.go) can change them while the simulation runs
var REFACTORY_PERIOD = 7.0

// Fraction of the refactory period that is fully blocked (absolute refactory period) before the block falls away in a bell shaped tail (relative refactory period)
var REFACTORY_PEAK_FRACTION = 0.5

var ACTIVATION_DECAY_RATE = 0.98

// Lookup tables for the per cycle decays, so they don't need math.Pow
var activationDecayPow = NewPowTable(ACTIVATION_DECAY_RATE)
var thresholdDecayPow = NewPowTable(THRESHOLD_DECAY_RATE)

const THRESHOLD_DECAY_RATE = 0.99

// Neighbour synchrony: when a neuron fires, it counts how many of its 8 surrounding neurons fired recently
// Each neighbour counts as exp(-timeSinceFired / NEIGHBOUR_FIRE_WINDOW), so the count is continuous rather than a hard cutoff
// Firing alone rewards its contributing inputs, firing with ~2*NEIGHBOUR_SYNC_CROSSOVER neighbours fully punishes them, to force neighbours to differentiate
var NEIGHBOUR_FIRE_WINDOW = 2.0       // Time constant in cycles, a neighbour that fired this many cycles ago counts as ~0.37 of a neighbour
var NEIGHBOUR_SYNC_RATE = 0.5         // Lerp amount at full reward (0 neighbours) or full punish (2*crossover neighbours)
var NEIGHBOUR_SYNC_CROSSOVER = 1.5    // Number of recently fired neighbours where it switches from reward to punish
var NEIGHBOUR_SYNC_MAX_STRENGTH = 1.5 // Strength that a reward lerps towards, a punish lerps towards 0

// Quiet neuron bonus: the longer a neuron has gone without firing, the more its contributing inputs are rewarded when it finally fires
// Accumulates linearly as 1 + QUIET_REWARD_PER_CYCLE * cycles of silence, so a long silent neuron can be recruited back into the network
//
//	200 cycles   -> x1.2
//	1000 cycles  -> x2.0
//	3000 cycles  -> x4.0
var QUIET_REWARD_PER_CYCLE = 0.001 // Extra reward multiplier added for each cycle the neuron has gone without firing

// When there is a presynaptic fire then a postsynaptic fire with meaningful contribution, then it will reward the connection endrite this amount
var POST_SYNAPTIC_FIRE_REWARD = 0.1

// Neuron represents a single neuron in the network.
type Neuron struct {
	Brain        *Brain
	ParentSquare *BrainSquare
	Index        uint32 // Position in Brain.Neurons
	Incoming     []IncomingConnection
	OutStart     uint32 // Outgoing connections are Brain.Connections[OutStart:OutEnd]
	OutEnd       uint32

	// Basic state (can be expanded)
	LastfireTime              float64
	AlreadyAddedToTocheckList bool
	IsExcitatory              bool

	Threshold float64

	RefactoryJitter float64 // Per neuron random extra on top of REFACTORY_PERIOD

	Activation     float64
	LastUpdateTime float64
}

// NewNeuron creates and returns a new neuron.
func NewNeuron(brain *Brain, parentSquare *BrainSquare, index uint32, isExcitatory bool) Neuron {
	return Neuron{
		Brain:                     brain,
		ParentSquare:              parentSquare,
		Index:                     index,
		Incoming:                  make([]IncomingConnection, 0),
		Activation:                rand.Float64()*0.5 + 0.5, //0.5 to 1
		LastfireTime:              0.0,
		LastUpdateTime:            0.0,
		Threshold:                 1.0 + ((rand.Float64() - 0.5) / 3), // Random threshold between 0.5 and 1.0
		RefactoryJitter:           rand.Float64() * 3,
		IsExcitatory:              isExcitatory,
		AlreadyAddedToTocheckList: false,
	}

}

// RefactoryPeriod is computed live so the REFACTORY_PERIOD slider affects existing neurons
func (n *Neuron) RefactoryPeriod() float64 {
	return REFACTORY_PERIOD + n.RefactoryJitter
}

// AddIncomingConnection records a connection (by index) from another neuron that sends signals *to* this neuron.
func (n *Neuron) AddIncomingConnection(conIndex uint32, from uint32) {
	n.Incoming = append(n.Incoming, IncomingConnection{Conn: conIndex, From: from})
}

func (n *Neuron) InitialConnect(brain *Brain, brainSquare *BrainSquare) {

	// This neuron's connections are appended to the end of brain.Connections, so they sit side by side
	n.OutStart = uint32(len(brain.Connections))

	if n.IsExcitatory {
		connectExcitatory(brain, brainSquare, n)
	} else {
		connectInhibitory(brain, n)
	}

	n.OutEnd = uint32(len(brain.Connections))

	// Shuffle outgoing, has to happen before the incoming indexes are recorded
	outgoing := brain.Connections[n.OutStart:n.OutEnd]
	rand.Shuffle(len(outgoing), func(i, j int) {
		outgoing[i], outgoing[j] = outgoing[j], outgoing[i]
	})

	for i := range outgoing {
		brain.Neurons[outgoing[i].To].AddIncomingConnection(n.OutStart+uint32(i), n.Index)
	}
}

func (n *Neuron) CheckForFiring(time float64) {

	n.UpdateActivationAndWeightOnly(time)

	// Recalculate threshold
	var lastFireTimeDelta = time - n.LastfireTime
	var desireToFireComp = -lastFireTimeDelta / 8000 //math.Min(lastFireTimeDelta/8000, 0.8)
	//var absoluteRefactoryPeriod = REFACTORY_PERIOD * 2
	//var justFiredDepolarization = math.Pow(math.Max(absoluteRefactoryPeriod-lastFireTimeDelta, 0)/absoluteRefactoryPeriod, 0.5) * 3 // Multiply the SPIKE_DURATION by 2 to get teh absolute refactory period
	virtualThreshold := n.Threshold + desireToFireComp // + justFiredDepolarization

	// No hard refactory check here, incoming activation is scaled down by the refactory block instead so only a big enough input can fire it during the tail
	if n.Activation >= virtualThreshold { // SUCCESSFUL FIRE!!!

		n.Threshold = Lerp(n.Threshold, 1.1, 0.3) // Threshold is always falling so this brings it back up to 'normal'

		// Because the simulation is in timesteps, it can be overshot, whereas in nature, as soon at the spike occurs, it will weaken any more

		//var overShootThreshold = 1.5 // Need to add 0.5 because a single normal fire may overshoot naturally
		var overshoot = math.Max(n.Activation-1.2, 0.0)
		hasOvershot := overshoot > 0
		//var overshootCorrector = math.Tanh(overshoot/2.0) * 0.3

		neurons := n.Brain.Neurons
		connections := n.Brain.Connections

		overShootWeightsTotal := 0.0
		if hasOvershot {
			for _, in := range n.Incoming {
				c := &connections[in.Conn]

				if c.IsAwaitingReward {
					var conFiredDurationAgo = time - neurons[in.From].LastfireTime

					if conFiredDurationAgo <= 1.0001 {
						if n.IsExcitatory {
							//c.Strength *= 1.0 - (overshoot / 5.0)
							overShootWeightsTotal += c.Strength
						}
					}
				}
			}
		}

		// Signed sync amount from the soft count of recently fired neighbours, +1 is full reward, -1 is full punish
		// With the default crossover of 1.5:
		//   0 neighbours   -> +1   (fired alone, full reward)
		//   1 neighbour    -> +0.33
		//   1.5 neighbours -> 0    (neutral)
		//   2 neighbours   -> -0.33
		//   3+ neighbours  -> -1   (full punish)
		neighbourSync := 1.0 - n.RecentlyFiredNeighbourCount(time)/NEIGHBOUR_SYNC_CROSSOVER
		neighbourSync = math.Max(-1.0, math.Min(1.0, neighbourSync))

		neighbourSyncAmount := math.Abs(neighbourSync) * NEIGHBOUR_SYNC_RATE

		// Reward multiplier for how long this neuron was silent, LastfireTime hasn't been reset yet so lastFireTimeDelta is the silence
		quietRewardBonus := 1.0 + QUIET_REWARD_PER_CYCLE*lastFireTimeDelta

		// Send reward or punsih to presynaptic dendrites
		for _, in := range n.Incoming {
			c := &connections[in.Conn]
			c.UpdateWeightsSince(time)
			var conFiredDurationAgo = time - neurons[in.From].LastfireTime

			if c.IsAwaitingReward {

				strength := c.Strength

				// Rewards both excitory and inhoibotory neurons
				contribution := activationDecayPow.Pow(ACTIVATION_DECAY_RATE, conFiredDurationAgo)

				idealStrength := contribution * quietRewardBonus
				// Step towards the ideal by a fixed amount plus a fraction of the gap, capped so it never overshoots the ideal
				delta := idealStrength - strength
				step := math.Min(math.Abs(delta), POST_SYNAPTIC_FIRE_REWARD*math.Abs(delta)+POST_SYNAPTIC_FIRE_REWARD)
				strength += math.Copysign(step, delta)

				//strength = RewardValue(strength, 0.06*contribution*quietRewardBonus, 1.5)

				if hasOvershot && conFiredDurationAgo <= 1.0001 {
					if n.IsExcitatory {
						relativeConnectionRatio := strength / overShootWeightsTotal
						strength *= math.Max(1.0-(relativeConnectionRatio*(overshoot/2.0)), 0.0) // Ratio can exceed 1, so clamp to stop it going negative
					}
				}

				// Reward inputs that drove this neuron to fire alone, punish ones that drove it to fire with its neighbours
				strength = math.Max(Lerp(strength, NEIGHBOUR_SYNC_MAX_STRENGTH, neighbourSyncAmount), 0.00001)

				c.Strength = strength
				c.IsAwaitingReward = false
			}
		}

		// Send signal to postsynaptic neurons
		n.LastfireTime = time
		n.Activation = 0
		outgoing := connections[n.OutStart:n.OutEnd]
		for i := range outgoing {
			c := &outgoing[i]
			to := &neurons[c.To]

			c.UpdateWeightsSince(time)
			to.UpdateActivationAndWeightOnly(time)

			// Refactory block is how much incoming activation is blocked, 1 is fully blocked and 0 is fully open
			// It holds at 1 for the peak of the refactory period, then falls away along the tail of a bell to 0 at the end of the period
			// so a big enough incoming spike late in the refactory period can still cause a fire
			//
			//	|‾‾‾‾‾\
			//	|      \
			//	|       ‾‾‾____
			//	0   peak       period
			timeSinceLastFire := time - to.LastfireTime
			refactoryPeriod := to.RefactoryPeriod()
			refactoryPeak := refactoryPeriod * REFACTORY_PEAK_FRACTION

			refactoryBlock := 0.0
			if timeSinceLastFire <= refactoryPeak {
				refactoryBlock = 1.0
			} else if timeSinceLastFire < refactoryPeriod {
				x := (timeSinceLastFire - refactoryPeak) / (refactoryPeriod - refactoryPeak) // 0 at the peak, 1 at the end of the period
				refactoryBlock = CosineFall(x)
			}
			passThrough := 1.0 - refactoryBlock

			strength := c.Strength

			if n.IsExcitatory { // All of a neuron's outgoing connections share its type

				if refactoryBlock > 0 { // If it has fired into a already spiked cell, weaken by how deep into the refactory period it is
					strength = PunishValue(strength, 0.05*refactoryBlock)
				}

				if passThrough > 0 { // Normal firing, reduced by the refactory block
					to.Activation += strength * passThrough // * PosOnlyRandMultiplier(strength*0.001) // We could add a magnitude multiplyer to the cell, default to 0.15 to allow the growth to always be between 0 and 1
					c.IsAwaitingReward = true
				}

				// Because we have added activation to the next neuron we will check if it has fired
				if !to.AlreadyAddedToTocheckList {
					to.AlreadyAddedToTocheckList = true
					n.Brain.NextNeuronsToCheck = append(n.Brain.NextNeuronsToCheck, to)
				}

			} else { // negative neuron outgoing

				to.Activation -= strength * passThrough
			}

			c.Strength = strength * 0.999 //Tiny amount of punish to stop rapid firing
		}
	}
}

// RecentlyFiredNeighbourCount is a soft count of the neurons in the 8 surrounding squares that fired recently
// Each neighbour adds exp(-timeSinceFired / NEIGHBOUR_FIRE_WINDOW), so firing right now counts as 1 and it fades away over time
func (n *Neuron) RecentlyFiredNeighbourCount(time float64) float64 {

	var sourceX = n.ParentSquare.PosX
	var sourceY = n.ParentSquare.PosY

	count := 0.0

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
			count += math.Exp(-(time - neighbour.LastfireTime) / NEIGHBOUR_FIRE_WINDOW)
		}
	}

	return count
}

func (n *Neuron) UpdateActivationAndWeightOnly(time float64) {

	if time == n.LastUpdateTime {
		return
	}

	var cyclesSince = time - n.LastUpdateTime

	n.Activation *= activationDecayPow.Pow(ACTIVATION_DECAY_RATE, cyclesSince)

	n.Threshold *= thresholdDecayPow.Pow(THRESHOLD_DECAY_RATE, cyclesSince)

	n.LastUpdateTime = time
}

func (n *Neuron) AddActivationInput(amount float64, time float64) {
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
					To: neuronToConnectTo.Index,
				}
				con.SetDistanceBase(sourceDist)
				con.Strength = float64(con.BaseStrength) * (0.75 + rand.Float64()*0.5) // Start within 25% of the base
				con.LongTermStrength = con.Strength

				// Incoming is recorded by InitialConnect once this neuron's block is shuffled
				brain.Connections = append(brain.Connections, con)
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
					To:            neuronToConnectTo.Index,
					Strength:      rand.Float64() * 0.05,
					BaseStrength:  0.0, // Inhibitory connections still decay towards 0
					LogRelaxDecay: float32(math.Log(0.999)),
				}
				con.LongTermStrength = con.Strength

				// Incoming is recorded by InitialConnect once this neuron's block is shuffled
				brain.Connections = append(brain.Connections, con)

				numberOutgoing += 1
			}
		}
	}

	//fmt.Println("Number outgoing: ", numberOutgoing)
}
