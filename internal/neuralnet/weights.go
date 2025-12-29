package neuralnet

import (
	"encoding/binary"
	"fmt"
	"io"
	"math"
	"os"
)

// Binary weights file constants
const (
	WeightsMagicBinary   = 472.3782 // Magic number for binary weights file
	WeightsVersionBinary = 1.01     // Expected version
)

// Weights contains all the neural networks used for position evaluation
type Weights struct {
	Contact  *NeuralNet // Contact position evaluation
	Race     *NeuralNet // Race position evaluation
	Crashed  *NeuralNet // Crashed position evaluation
	PContact *NeuralNet // Pruning net for contact
	PCrashed *NeuralNet // Pruning net for crashed
	PRace    *NeuralNet // Pruning net for race
}

// LoadWeightsBinary loads all neural networks from a binary weights file (gnubg.wd)
func LoadWeightsBinary(path string) (*Weights, error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, fmt.Errorf("opening weights file: %w", err)
	}
	defer f.Close()

	return LoadWeightsBinaryFromReader(f)
}

// LoadWeightsBinaryFromReader loads all neural networks from a reader
func LoadWeightsBinaryFromReader(r io.Reader) (*Weights, error) {
	// Read and validate magic number header
	var magic, version float32
	if err := binary.Read(r, binary.LittleEndian, &magic); err != nil {
		return nil, fmt.Errorf("reading magic number: %w", err)
	}
	if math.Abs(float64(magic)-WeightsMagicBinary) > 0.001 {
		return nil, fmt.Errorf("invalid magic number: %f (expected %f)", magic, WeightsMagicBinary)
	}

	if err := binary.Read(r, binary.LittleEndian, &version); err != nil {
		return nil, fmt.Errorf("reading version: %w", err)
	}
	if version < 1.0 || version > 2.0 {
		return nil, fmt.Errorf("unsupported weights version: %f", version)
	}

	w := &Weights{}
	var err error

	// Load in the same order as gnubg
	w.Contact, err = LoadBinary(r)
	if err != nil {
		return nil, fmt.Errorf("loading contact net: %w", err)
	}

	w.Race, err = LoadBinary(r)
	if err != nil {
		return nil, fmt.Errorf("loading race net: %w", err)
	}

	w.Crashed, err = LoadBinary(r)
	if err != nil {
		return nil, fmt.Errorf("loading crashed net: %w", err)
	}

	w.PContact, err = LoadBinary(r)
	if err != nil {
		return nil, fmt.Errorf("loading pruning contact net: %w", err)
	}

	w.PCrashed, err = LoadBinary(r)
	if err != nil {
		return nil, fmt.Errorf("loading pruning crashed net: %w", err)
	}

	w.PRace, err = LoadBinary(r)
	if err != nil {
		return nil, fmt.Errorf("loading pruning race net: %w", err)
	}

	return w, nil
}

// LoadWeightsText loads all neural networks from a text weights file (gnubg.weights)
func LoadWeightsText(path string) (*Weights, error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, fmt.Errorf("opening weights file: %w", err)
	}
	defer f.Close()

	return LoadWeightsTextFromReader(f)
}

// LoadWeightsTextFromReader loads all neural networks from a text reader
func LoadWeightsTextFromReader(r io.Reader) (*Weights, error) {
	// Skip the header line "GNU Backgammon X.XX"
	var h1, h2, h3 string
	if _, err := fmt.Fscanln(r, &h1, &h2, &h3); err != nil {
		return nil, fmt.Errorf("reading header: %w", err)
	}

	w := &Weights{}
	var err error

	// Load in the same order as gnubg
	w.Contact, err = LoadText(r)
	if err != nil {
		return nil, fmt.Errorf("loading contact net: %w", err)
	}

	w.Race, err = LoadText(r)
	if err != nil {
		return nil, fmt.Errorf("loading race net: %w", err)
	}

	w.Crashed, err = LoadText(r)
	if err != nil {
		return nil, fmt.Errorf("loading crashed net: %w", err)
	}

	w.PContact, err = LoadText(r)
	if err != nil {
		return nil, fmt.Errorf("loading pruning contact net: %w", err)
	}

	w.PCrashed, err = LoadText(r)
	if err != nil {
		return nil, fmt.Errorf("loading pruning crashed net: %w", err)
	}

	w.PRace, err = LoadText(r)
	if err != nil {
		return nil, fmt.Errorf("loading pruning race net: %w", err)
	}

	return w, nil
}

// Validate checks that the loaded weights have the expected dimensions
func (w *Weights) Validate() error {
	// Expected dimensions based on gnubg constants
	const (
		numOutputs       = 5
		numPruningInputs = 200 // 25 * 4 * 2
	)

	// Contact and Crashed nets have the same input size
	// NUM_INPUTS = (25 * 4 + MORE_INPUTS) * 2
	// MORE_INPUTS = 25 (from the enum)
	// So NUM_INPUTS = (100 + 25) * 2 = 250
	// Actually looking at the code: NUM_INPUTS = (25 * MINPPERPOINT + MORE_INPUTS) * 2
	// MINPPERPOINT = 4, MORE_INPUTS = 25
	// NUM_INPUTS = (25 * 4 + 25) * 2 = (100 + 25) * 2 = 250

	if w.Contact.COutput != numOutputs {
		return fmt.Errorf("contact net has %d outputs, expected %d", w.Contact.COutput, numOutputs)
	}
	if w.Race.COutput != numOutputs {
		return fmt.Errorf("race net has %d outputs, expected %d", w.Race.COutput, numOutputs)
	}
	if w.Crashed.COutput != numOutputs {
		return fmt.Errorf("crashed net has %d outputs, expected %d", w.Crashed.COutput, numOutputs)
	}

	// Pruning nets should have 200 inputs (base inputs only)
	if w.PContact.CInput != numPruningInputs {
		return fmt.Errorf("pruning contact net has %d inputs, expected %d", w.PContact.CInput, numPruningInputs)
	}
	if w.PCrashed.CInput != numPruningInputs {
		return fmt.Errorf("pruning crashed net has %d inputs, expected %d", w.PCrashed.CInput, numPruningInputs)
	}
	if w.PRace.CInput != numPruningInputs {
		return fmt.Errorf("pruning race net has %d inputs, expected %d", w.PRace.CInput, numPruningInputs)
	}

	return nil
}

// String returns a summary of the loaded weights
func (w *Weights) String() string {
	return fmt.Sprintf("Weights{\n"+
		"  Contact:  %d -> %d -> %d\n"+
		"  Race:     %d -> %d -> %d\n"+
		"  Crashed:  %d -> %d -> %d\n"+
		"  PContact: %d -> %d -> %d\n"+
		"  PCrashed: %d -> %d -> %d\n"+
		"  PRace:    %d -> %d -> %d\n"+
		"}",
		w.Contact.CInput, w.Contact.CHidden, w.Contact.COutput,
		w.Race.CInput, w.Race.CHidden, w.Race.COutput,
		w.Crashed.CInput, w.Crashed.CHidden, w.Crashed.COutput,
		w.PContact.CInput, w.PContact.CHidden, w.PContact.COutput,
		w.PCrashed.CInput, w.PCrashed.CHidden, w.PCrashed.COutput,
		w.PRace.CInput, w.PRace.CHidden, w.PRace.COutput,
	)
}
