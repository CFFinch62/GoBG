package neuralnet

import (
	"path/filepath"
	"testing"
)

// Global to prevent compiler optimizations
var benchOutput []float32

func loadBenchmarkWeights(b *testing.B) *Weights {
	weightsPath := filepath.Join("..", "..", "data", "gnubg.weights")
	w, err := LoadWeightsText(weightsPath)
	if err != nil {
		b.Fatalf("Failed to load weights: %v", err)
	}
	return w
}

func BenchmarkEvaluateOriginal(b *testing.B) {
	w := loadBenchmarkWeights(b)

	// Create a contact position input (250 inputs)
	input := make([]float32, 250)
	for i := range input {
		if i%4 == 0 {
			input[i] = 1.0
		} else if i%7 == 0 {
			input[i] = 0.5
		}
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		benchOutput = w.Contact.Evaluate(input)
	}
}

func BenchmarkEvaluateSIMD(b *testing.B) {
	w := loadBenchmarkWeights(b)

	// Create a contact position input (250 inputs)
	input := make([]float32, 250)
	for i := range input {
		if i%4 == 0 {
			input[i] = 1.0
		} else if i%7 == 0 {
			input[i] = 0.5
		}
	}

	// Pre-allocate buffer
	buf := NewEvaluateBuffer(w.Contact.CInput, w.Contact.CHidden)
	output := make([]float32, w.Contact.COutput)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		w.Contact.EvaluateSIMD(input, output, buf)
		benchOutput = output
	}
}

func BenchmarkEvaluateInto(b *testing.B) {
	w := loadBenchmarkWeights(b)

	// Create a contact position input (250 inputs)
	input := make([]float32, 250)
	for i := range input {
		if i%4 == 0 {
			input[i] = 1.0
		} else if i%7 == 0 {
			input[i] = 0.5
		}
	}

	output := make([]float32, w.Contact.COutput)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		w.Contact.EvaluateInto(input, output)
		benchOutput = output
	}
}

func BenchmarkSigmoidOriginal(b *testing.B) {
	x := float32(-2.5)
	var result float32
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		result = sigmoid(x)
	}
	_ = result
}

func BenchmarkSigmoidFast(b *testing.B) {
	x := float32(-2.5)
	var result float32
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		result = sigmoidFast(x)
	}
	_ = result
}

func BenchmarkRaceEvaluate(b *testing.B) {
	w := loadBenchmarkWeights(b)

	// Create a race position input (214 inputs)
	input := make([]float32, 214)
	for i := range input {
		if i%4 == 0 {
			input[i] = 1.0
		}
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		benchOutput = w.Race.Evaluate(input)
	}
}

func BenchmarkRaceEvaluateSIMD(b *testing.B) {
	w := loadBenchmarkWeights(b)

	// Create a race position input (214 inputs)
	input := make([]float32, 214)
	for i := range input {
		if i%4 == 0 {
			input[i] = 1.0
		}
	}

	buf := NewEvaluateBuffer(w.Race.CInput, w.Race.CHidden)
	output := make([]float32, w.Race.COutput)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		w.Race.EvaluateSIMD(input, output, buf)
		benchOutput = output
	}
}

// Test that SIMD produces same results as original
func TestSIMDAccuracy(t *testing.T) {
	w := loadBenchmarkWeights((*testing.B)(nil))
	if w == nil {
		t.Skip("Could not load weights")
	}

	input := make([]float32, 250)
	for i := range input {
		if i%4 == 0 {
			input[i] = 1.0
		} else if i%7 == 0 {
			input[i] = 0.5
		}
	}

	original := w.Contact.Evaluate(input)
	simd := w.Contact.EvaluateSIMDSimple(input)

	for i := range original {
		diff := original[i] - simd[i]
		if diff < 0 {
			diff = -diff
		}
		if diff > 0.001 {
			t.Errorf("Output %d differs: original=%f, simd=%f, diff=%f", i, original[i], simd[i], diff)
		}
	}
}

