package main

import (
	"math"
	"math/rand"
)

const CONTRIBUTION_REWARD_WINDOW = 20.0

const SPIKE_DURATION = 5.0

// Neuron represents a single neuron in the network.
type Neuron struct {
	ThreadStrip  *MultithreadStrip
	ParentSquare *BrainSquare
	Incoming     []*Connection
	Outgoing     []*Connection

	// Basic state (can be expanded)
	LastfireTime              float64
	AlreadyAddedToTocheckList bool
	IsExcitatory              bool

	Threshold float64

	Activation     float64
	LastUpdateTime float64
}

// NewNeuron creates and returns a new neuron.
func NewNeuron(x, y int, parentSquare *BrainSquare, isExcitatory bool) Neuron {
	return Neuron{
		ParentSquare:              parentSquare,
		Incoming:                  make([]*Connection, 0),
		Outgoing:                  make([]*Connection, 0),
		Activation:                rand.Float64()*0.5 + 0.5, //0.5 to 1
		LastfireTime:              0.0,
		LastUpdateTime:            0.0,
		Threshold:                 1.0, // Random threshold between 0.5 and 1.0
		IsExcitatory:              isExcitatory,
		AlreadyAddedToTocheckList: false,
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

	n.UpdateActivationOnly(time)

	// Recalculate threshold
	var lastFireTimeDelta = time - n.LastfireTime
	var desireToFireComp = -math.Min(lastFireTimeDelta/8000, 0.99)
	var absoluteRefactoryPeriod = SPIKE_DURATION * 2
	var justFiredDepolarization = math.Pow(math.Max(absoluteRefactoryPeriod-lastFireTimeDelta, 0)/absoluteRefactoryPeriod, 0.5) * 3 // Multiply the SPIKE_DURATION by 2 to get teh absolute refactory period
	n.Threshold = 1.0 + desireToFireComp + justFiredDepolarization

	if n.Activation >= n.Threshold { // SUCCESSFUL FIRE!!!

		// Because the simulation is in timesteps, it can be overshot, whereas in nature, as soon at the spike occurs, it will weaken any more

		var overShootThreshold = 1.5 // Need to add 0.5 because a single normal fire may overshoot naturally
		var overshoot = math.Max(n.Activation-overShootThreshold, 0.0)
		var overshootCorrector = math.Tanh(overshoot/2.0) * 0.3

		// Send reward or punsih to presynaptic dendrites
		for _, c := range n.Incoming {
			var firedDurationAgo = time - c.From.LastfireTime
			if c.IsAwaitingReward && firedDurationAgo < CONTRIBUTION_REWARD_WINDOW {

				// The closer to the fireing time the more reward
				var changeAmount = 0.2 + (CONTRIBUTION_REWARD_WINDOW-firedDurationAgo)/CONTRIBUTION_REWARD_WINDOW

				// more reward to the connection if this neuron hasnt fired in a while
				changeAmount += math.Pow(lastFireTimeDelta/5000, 2)

				if c.IsExcitatory {
					if overshoot > 0.001 {
						//c.DebugLast = "Overshoot by " + fmt.Sprintf("%f", overshootMultiplier)
						c.Strength = PunishValue(c.Strength, overshootCorrector) //TOO Punishing in certain circumstances
					} else {
						c.Strength = RewardValue(c.Strength, 0.07*changeAmount, 1.5)
					}
				} else { // Negative
					//c.DebugLast = "Negative, postsynaptic fired"
					c.Strength = PunishValue(c.Strength, 0.04*changeAmount)
				}

			} else { // Punish the connection because it wasn't involved in the cause of its firing
				if c.IsExcitatory {
					//c.DebugLast = "Wasnt involved in firing"
					c.Strength = PunishValue(c.Strength, 0.005*c.Strength)
				}
			}
			c.IsAwaitingReward = false
		}

		// Send signal to postsynaptic neurons
		n.LastfireTime = time
		n.Activation = 0
		for _, c := range n.Outgoing {

			if c.IsExcitatory {

				c.To.UpdateActivationOnly(time)

				if time-c.To.LastfireTime < SPIKE_DURATION { // If it has fired into a already spiked cell, weaken
					c.IsAwaitingReward = false
					//c.DebugLast = "Fired into spike"
					c.Strength = PunishValue(c.Strength, 0.03)
				} else { // Normal firing, punish up front for rewward later
					c.To.Activation += c.Strength // * PosOnlyRandMultiplier(c.Strength*0.001) // We could add a magnitude multiplyer to the cell, default to 0.15 to allow the growth to always be between 0 and 1
					c.IsAwaitingReward = true
					//c.DebugLast = "Normal firing"
					c.Strength = PunishValue(c.Strength, 0.008)
				}

				// Because we have added activation to the next neuron we will check if it has fired
				if !c.To.AlreadyAddedToTocheckList {
					c.To.AlreadyAddedToTocheckList = true
					c.To.ThreadStrip.NextNeuronsToCheck = append(c.To.ThreadStrip.NextNeuronsToCheck, c.To)
				}

			} else { // negative neuron outgoing

				c.To.UpdateActivationOnly(time)

				// We want reward if its firing into a neuron with much activation, and punish if it fires into a inactive neuron, proportionally
				var existingActivation = c.To.Activation
				var reward = existingActivation - 0.4 // Only reward if the activation is above 0.4

				c.To.Activation -= c.Strength

				if reward > 0 {
					c.IsAwaitingReward = true
					c.Strength = RewardValue(c.Strength, reward*0.1, 1.5) // Should only reward based on the amount of activation there
				} else { // Reward is always negative here
					//c.DebugLast = "Punish negative for firing onto no activation"
					c.Strength = PunishValue(c.Strength, -reward*0.05*c.Strength)
				}
			}
		}
	} else if n.Activation > n.Threshold*0.8 { // Nearly fired
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
					c.Strength = PunishValue(c.Strength, 0.1*rewardAmount*timingReward) // This needs to be belenced with the amount of upfront punish
				}
			}
		}
	}
}

func (n *Neuron) UpdateActivationOnly(time float64) {

	if time == n.LastUpdateTime {
		return
	}

	var cyclesSince = time - n.LastUpdateTime

	n.Activation = n.Activation * math.Pow(0.98, cyclesSince)

	n.LastUpdateTime = time
}

func (n *Neuron) AddActivation(amount float64, time float64) {
	n.UpdateActivationOnly(time)
	n.Activation += amount
	if n.Activation < 0 {
		n.Activation = 0
	}
	n.CheckForFiring(time)
}

func connectExcitatory(brain *Brain, brainSquare *BrainSquare, n *Neuron) {

	var sourceX = n.ParentSquare.PosX
	var sourceY = n.ParentSquare.PosY

	var radiusI = 5
	var radiusF = float64(radiusI)

	var numberOutgoing = 0

	// Not sure why these are flipped?
	var circlePositionX = int(float64(sourceX) + float64(brainSquare.DirVecX*(radiusF+1.0)))
	var circlePositionY = int(float64(sourceY) + float64(brainSquare.DirVecY*(radiusF+1.0)))

	for y := -radiusI; y <= radiusI; y++ {
		for x := -radiusI; x <= radiusI; x++ {

			var dist = math.Sqrt(math.Pow(float64(x), 2) + math.Pow(float64(y), 2))

			if dist > radiusF {
				continue
			}

			mappedPosX := x + circlePositionX
			mappedPosY := y + circlePositionY

			if mappedPosX == sourceX && mappedPosY == sourceY {
				continue // Dont connect to self
			}

			wrappedX, wrappedY := brain.WrapPosition(mappedPosX, mappedPosY)

			targetNeuronsSlice := brain.BrainSquares[wrappedX][wrappedY].Neurons

			for i := 0; i < len(targetNeuronsSlice); i++ {
				neuronToConnectTo := &targetNeuronsSlice[i]

				var con = Connection{
					From:         n,
					To:           neuronToConnectTo,
					IsExcitatory: true,
					Strength:     rand.Float64()*0.1 + 0.1,
				}

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
				}

				n.AddOutgoingConnection(&con)
				neuronToConnectTo.AddIncomingConnection(&con)

				numberOutgoing += 1
			}
		}
	}

	//fmt.Println("Number outgoing: ", numberOutgoing)
}
