// Package neuralnet implements neural network evaluation for backgammon positions.
// This is a port of gnubg's neuralnet.c
package neuralnet

import (
	"encoding/binary"
	"fmt"
	"io"
	"math"
)

// NeuralNet represents a trained neural network for position evaluation
type NeuralNet struct {
	CInput          uint32    // Number of input nodes
	CHidden         uint32    // Number of hidden nodes
	COutput         uint32    // Number of output nodes
	NTrained        int32     // Training status
	RBetaHidden     float32   // Beta for hidden layer sigmoid
	RBetaOutput     float32   // Beta for output layer sigmoid
	HiddenWeight    []float32 // Weights from input to hidden layer
	OutputWeight    []float32 // Weights from hidden to output layer
	HiddenThreshold []float32 // Thresholds for hidden nodes
	OutputThreshold []float32 // Thresholds for output nodes
}

// Evaluate computes the neural network output for the given input
func (nn *NeuralNet) Evaluate(input []float32) []float32 {
	output := make([]float32, nn.COutput)
	nn.EvaluateInto(input, output)
	return output
}

// EvaluateInto computes the neural network output into the provided slice
func (nn *NeuralNet) EvaluateInto(input, output []float32) {
	// Allocate hidden layer activations
	ar := make([]float32, nn.CHidden)

	// Initialize with thresholds
	copy(ar, nn.HiddenThreshold)

	// Calculate activity at hidden nodes
	prWeight := 0
	for i := uint32(0); i < nn.CInput; i++ {
		ari := input[i]
		if ari == 0.0 {
			prWeight += int(nn.CHidden)
		} else if ari == 1.0 {
			for j := uint32(0); j < nn.CHidden; j++ {
				ar[j] += nn.HiddenWeight[prWeight]
				prWeight++
			}
		} else {
			for j := uint32(0); j < nn.CHidden; j++ {
				ar[j] += nn.HiddenWeight[prWeight] * ari
				prWeight++
			}
		}
	}

	// Apply sigmoid to hidden layer
	for i := uint32(0); i < nn.CHidden; i++ {
		ar[i] = sigmoid(-nn.RBetaHidden * ar[i])
	}

	// Calculate activity at output nodes
	prWeight = 0
	for i := uint32(0); i < nn.COutput; i++ {
		r := nn.OutputThreshold[i]
		for j := uint32(0); j < nn.CHidden; j++ {
			r += ar[j] * nn.OutputWeight[prWeight]
			prWeight++
		}
		output[i] = sigmoid(-nn.RBetaOutput * r)
	}
}

// LoadBinary loads a neural network from a binary file
func LoadBinary(r io.Reader) (*NeuralNet, error) {
	nn := &NeuralNet{}

	// Read header
	if err := binary.Read(r, binary.LittleEndian, &nn.CInput); err != nil {
		return nil, fmt.Errorf("reading cInput: %w", err)
	}
	if err := binary.Read(r, binary.LittleEndian, &nn.CHidden); err != nil {
		return nil, fmt.Errorf("reading cHidden: %w", err)
	}
	if err := binary.Read(r, binary.LittleEndian, &nn.COutput); err != nil {
		return nil, fmt.Errorf("reading cOutput: %w", err)
	}
	if err := binary.Read(r, binary.LittleEndian, &nn.NTrained); err != nil {
		return nil, fmt.Errorf("reading nTrained: %w", err)
	}
	if err := binary.Read(r, binary.LittleEndian, &nn.RBetaHidden); err != nil {
		return nil, fmt.Errorf("reading rBetaHidden: %w", err)
	}
	if err := binary.Read(r, binary.LittleEndian, &nn.RBetaOutput); err != nil {
		return nil, fmt.Errorf("reading rBetaOutput: %w", err)
	}

	// Validate
	if nn.CInput < 1 || nn.CHidden < 1 || nn.COutput < 1 {
		return nil, fmt.Errorf("invalid network dimensions: %d/%d/%d", nn.CInput, nn.CHidden, nn.COutput)
	}
	if nn.RBetaHidden <= 0 || nn.RBetaOutput <= 0 {
		return nil, fmt.Errorf("invalid beta values: %f/%f", nn.RBetaHidden, nn.RBetaOutput)
	}

	// Allocate and read weights
	nn.HiddenWeight = make([]float32, nn.CInput*nn.CHidden)
	if err := binary.Read(r, binary.LittleEndian, nn.HiddenWeight); err != nil {
		return nil, fmt.Errorf("reading hidden weights: %w", err)
	}

	nn.OutputWeight = make([]float32, nn.CHidden*nn.COutput)
	if err := binary.Read(r, binary.LittleEndian, nn.OutputWeight); err != nil {
		return nil, fmt.Errorf("reading output weights: %w", err)
	}

	nn.HiddenThreshold = make([]float32, nn.CHidden)
	if err := binary.Read(r, binary.LittleEndian, nn.HiddenThreshold); err != nil {
		return nil, fmt.Errorf("reading hidden thresholds: %w", err)
	}

	nn.OutputThreshold = make([]float32, nn.COutput)
	if err := binary.Read(r, binary.LittleEndian, nn.OutputThreshold); err != nil {
		return nil, fmt.Errorf("reading output thresholds: %w", err)
	}

	return nn, nil
}

// sigmoid computes the sigmoid function 1 / (1 + e^x)
// This is an optimized approximation matching gnubg's implementation
func sigmoid(x float32) float32 {
	return float32(1.0 / (1.0 + math.Exp(float64(x))))
}

// LoadText loads a neural network from a text file format
func LoadText(r io.Reader) (*NeuralNet, error) {
	nn := &NeuralNet{}

	var dummy string
	_, err := fmt.Fscanf(r, "%d %d %d %s %f %f\n",
		&nn.CInput, &nn.CHidden, &nn.COutput, &dummy, &nn.RBetaHidden, &nn.RBetaOutput)
	if err != nil {
		return nil, fmt.Errorf("reading header: %w", err)
	}

	// Validate
	if nn.CInput < 1 || nn.CHidden < 1 || nn.COutput < 1 {
		return nil, fmt.Errorf("invalid network dimensions: %d/%d/%d", nn.CInput, nn.CHidden, nn.COutput)
	}
	if nn.RBetaHidden <= 0 || nn.RBetaOutput <= 0 {
		return nil, fmt.Errorf("invalid beta values: %f/%f", nn.RBetaHidden, nn.RBetaOutput)
	}

	nn.NTrained = 1

	// Allocate and read weights
	nn.HiddenWeight = make([]float32, nn.CInput*nn.CHidden)
	for i := range nn.HiddenWeight {
		if _, err := fmt.Fscanf(r, "%f\n", &nn.HiddenWeight[i]); err != nil {
			return nil, fmt.Errorf("reading hidden weight %d: %w", i, err)
		}
	}

	nn.OutputWeight = make([]float32, nn.CHidden*nn.COutput)
	for i := range nn.OutputWeight {
		if _, err := fmt.Fscanf(r, "%f\n", &nn.OutputWeight[i]); err != nil {
			return nil, fmt.Errorf("reading output weight %d: %w", i, err)
		}
	}

	nn.HiddenThreshold = make([]float32, nn.CHidden)
	for i := range nn.HiddenThreshold {
		if _, err := fmt.Fscanf(r, "%f\n", &nn.HiddenThreshold[i]); err != nil {
			return nil, fmt.Errorf("reading hidden threshold %d: %w", i, err)
		}
	}

	nn.OutputThreshold = make([]float32, nn.COutput)
	for i := range nn.OutputThreshold {
		if _, err := fmt.Fscanf(r, "%f\n", &nn.OutputThreshold[i]); err != nil {
			return nil, fmt.Errorf("reading output threshold %d: %w", i, err)
		}
	}

	return nn, nil
}
