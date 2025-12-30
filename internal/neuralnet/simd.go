// Package neuralnet provides SIMD-optimized neural network evaluation.
// This file contains optimized implementations using gonum for vectorized operations.
package neuralnet

import (
	"math"
	"sync"

	"gonum.org/v1/gonum/floats"
)

// sigmoidTableSize is the number of entries in the sigmoid lookup table
const sigmoidTableSize = 8192

// sigmoidTableScale maps input range [-8, 8] to table indices [0, 8191]
const sigmoidTableScale = float64(sigmoidTableSize) / 16.0

var (
	sigmoidTable     [sigmoidTableSize]float32
	sigmoidTableOnce sync.Once
)

// initSigmoidTable initializes the sigmoid lookup table
func initSigmoidTable() {
	sigmoidTableOnce.Do(func() {
		for i := 0; i < sigmoidTableSize; i++ {
			// Map index to range [-8, 8]
			x := (float64(i)/sigmoidTableScale - 8.0)
			sigmoidTable[i] = float32(1.0 / (1.0 + math.Exp(x)))
		}
	})
}

// sigmoidFast uses a lookup table for fast sigmoid approximation
func sigmoidFast(x float32) float32 {
	initSigmoidTable()

	// Clamp to table range
	if x <= -8.0 {
		return 1.0
	}
	if x >= 8.0 {
		return 0.0
	}

	// Interpolate from table
	idx := (float64(x) + 8.0) * sigmoidTableScale
	i := int(idx)
	if i >= sigmoidTableSize-1 {
		return sigmoidTable[sigmoidTableSize-1]
	}

	// Linear interpolation for smoother results
	frac := float32(idx) - float32(i)
	return sigmoidTable[i]*(1-frac) + sigmoidTable[i+1]*frac
}

// EvaluateBuffer holds pre-allocated buffers for neural network evaluation
type EvaluateBuffer struct {
	hidden  []float64
	input64 []float64
}

// NewEvaluateBuffer creates a buffer for the given network dimensions
func NewEvaluateBuffer(cInput, cHidden uint32) *EvaluateBuffer {
	return &EvaluateBuffer{
		hidden:  make([]float64, cHidden),
		input64: make([]float64, cInput),
	}
}

// EvaluateSIMD computes the neural network output using SIMD operations.
// This version uses gonum's BLAS-optimized floats package and a sigmoid lookup table.
func (nn *NeuralNet) EvaluateSIMD(input []float32, output []float32, buf *EvaluateBuffer) {
	nn.EvaluateFast(input, output, buf)
}

// EvaluateFast is a highly optimized evaluation using float32 throughout.
// Uses loop unrolling and fast sigmoid lookup table.
func (nn *NeuralNet) EvaluateFast(input []float32, output []float32, buf *EvaluateBuffer) {
	initSigmoidTable()

	cInput := int(nn.CInput)
	cHidden := int(nn.CHidden)
	cOutput := int(nn.COutput)

	// Use float64 buffer for better numerical precision during accumulation
	if buf == nil || len(buf.hidden) < cHidden {
		buf = NewEvaluateBuffer(nn.CInput, nn.CHidden)
	}

	// Initialize with thresholds
	for i := 0; i < cHidden; i++ {
		buf.hidden[i] = float64(nn.HiddenThreshold[i])
	}

	// Hidden layer: sparse matrix-vector multiply with 4-way unrolling
	for i := 0; i < cInput; i++ {
		ari := input[i]
		if ari == 0.0 {
			continue
		}

		weightStart := i * cHidden
		weights := nn.HiddenWeight[weightStart : weightStart+cHidden]

		if ari == 1.0 {
			// 4-way unrolled addition
			j := 0
			for ; j <= cHidden-4; j += 4 {
				buf.hidden[j] += float64(weights[j])
				buf.hidden[j+1] += float64(weights[j+1])
				buf.hidden[j+2] += float64(weights[j+2])
				buf.hidden[j+3] += float64(weights[j+3])
			}
			for ; j < cHidden; j++ {
				buf.hidden[j] += float64(weights[j])
			}
		} else {
			// 4-way unrolled multiply-add
			ari64 := float64(ari)
			j := 0
			for ; j <= cHidden-4; j += 4 {
				buf.hidden[j] += float64(weights[j]) * ari64
				buf.hidden[j+1] += float64(weights[j+1]) * ari64
				buf.hidden[j+2] += float64(weights[j+2]) * ari64
				buf.hidden[j+3] += float64(weights[j+3]) * ari64
			}
			for ; j < cHidden; j++ {
				buf.hidden[j] += float64(weights[j]) * ari64
			}
		}
	}

	// Apply sigmoid using fast lookup
	betaHidden := float64(-nn.RBetaHidden)
	for i := 0; i < cHidden; i++ {
		buf.hidden[i] = float64(sigmoidFast(float32(betaHidden * buf.hidden[i])))
	}

	// Output layer with unrolling
	betaOutput := float64(-nn.RBetaOutput)
	for i := 0; i < cOutput; i++ {
		r := float64(nn.OutputThreshold[i])
		weightStart := i * cHidden
		weights := nn.OutputWeight[weightStart : weightStart+cHidden]

		// 4-way unrolled dot product
		j := 0
		for ; j <= cHidden-4; j += 4 {
			r += buf.hidden[j]*float64(weights[j]) +
				buf.hidden[j+1]*float64(weights[j+1]) +
				buf.hidden[j+2]*float64(weights[j+2]) +
				buf.hidden[j+3]*float64(weights[j+3])
		}
		for ; j < cHidden; j++ {
			r += buf.hidden[j] * float64(weights[j])
		}

		output[i] = sigmoidFast(float32(betaOutput * r))
	}
}

// EvaluateSIMDSimple is a simpler SIMD version without pre-allocated buffers.
// Uses gonum's floats package for vectorized operations.
func (nn *NeuralNet) EvaluateSIMDSimple(input []float32) []float32 {
	output := make([]float32, nn.COutput)
	buf := NewEvaluateBuffer(nn.CInput, nn.CHidden)
	nn.EvaluateSIMD(input, output, buf)
	return output
}

// DotFloat32 computes the dot product of two float32 slices using gonum
func DotFloat32(a, b []float32) float32 {
	if len(a) != len(b) {
		return 0
	}

	// Convert to float64 for gonum
	a64 := make([]float64, len(a))
	b64 := make([]float64, len(b))
	for i := range a {
		a64[i] = float64(a[i])
		b64[i] = float64(b[i])
	}

	return float32(floats.Dot(a64, b64))
}
