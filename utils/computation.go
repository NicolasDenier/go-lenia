package utils

import (
	"math"
	"math/rand"
	"time"

	"gonum.org/v1/gonum/floats"
	"gonum.org/v1/gonum/mat"
)

// for random generation
var seed = rand.NewSource(time.Now().UnixNano())
var r0 = rand.New(seed)

type Config struct {
	// matrices
	A, Kernel, KFFT, G *mat.Dense
	// parameters
	R, T, Mu, Sigma, Dx, Dt float64
	Beta                    []float64
}

type compute interface {
	InitState()
	ComputeKernel()
	GrowthMapping()
	Update()
}

func fill(length int, value float64) []float64 {
	// create a new slice filled with the provided value
	grid := make([]float64, length)
	for i := range grid {
		grid[i] = value
	}
	return grid
}

func randInt(min, max int) int {
	return r0.Intn(max-min) + min
}

func Clip(n float64, min, max float64) float64 {
	// restrict a value between two bounds
	if n < min {
		return min
	} else if n > max {
		return max
	}
	return n
}

func (c *Config) InitState() {
	// define the initial state of A
	// for now, random rectangles
	h, w := c.A.Dims()
	// random number of rectagles
	for k := 0; k < randInt(10, 30); k++ {
		// random widths
		w1 := randInt(20, 50)
		w2 := randInt(20, 50)
		// center of rectangle position
		x := randInt(w1, w-w1)
		y := randInt(w2, h-w2)
		// fill the rectangle to 1
		for i := x - w1; i < x+w1; i++ {
			for j := y - w2; j < y+w2; j++ {
				c.A.Set(i, j, r0.Float64())
			}
		}
	}
}

func NewConfig(h, w int, R, T, Mu, Sigma float64, Beta []float64) Config {
	// create a new config with all variables initialized
	setup := Config{
		A: mat.NewDense(h, w, nil),
	}
	// set parameters
	setup.T = T
	setup.R = R
	setup.Mu = Mu
	setup.Sigma = Sigma
	setup.Dx = float64(1 / R)
	setup.Dt = float64(1 / T)
	setup.Beta = Beta
	// compute Kernel
	setup.ComputeKernel()
	// initialize A
	setup.InitState()
	return setup
}

func getRadiusMatrix(R int) *mat.Dense {
	// set the value of each pixel to be the distance to the center of the matrix
	m := mat.NewDense(2*R+1, 2*R+1, nil)
	for i := -R; i <= R; i++ {
		for j := -R; j <= R; j++ {
			distance := math.Sqrt(float64(i*i + j*j))
			m.Set(R+i, R+j, distance)
		}
	}
	return m
}

func KernelCore(r float64) float64 {
	var a float64 = 4
	value := math.Pow(4*r*(1-r), a)
	return value
}

func (c *Config) ComputeKernel() {
	// @radius = get_polar_radius_matrix(SIZE_X, SIZE_Y) * dx
	K := getRadiusMatrix(int(c.R))
	K.Scale(c.Dx, K)
	// @Br = size(beta) * @radius
	lenBeta := float64(len(c.Beta))
	K.Scale(lenBeta, K)
	// @kernel_shell = beta[floor(@Br)] * kernel_core(@Br % 1)
	KS := mat.DenseCopyOf(K)
	KS.Apply(func(_, _ int, v float64) float64 {
		// distance to the center over lenBeta is ignored
		if v >= lenBeta {
			return 0
		}
		return c.Beta[int(math.Floor(v))] * KernelCore(math.Mod(v, 1))
	}, K)
	// @kernel = @kernel_shell / sum(@kernel_shell)
	KS.Scale(1/floats.Sum(KS.RawMatrix().Data), KS)
	// return @kernel
	c.Kernel = mat.DenseCopyOf(KS)
}

func (c *Config) GrowthMapping(U *mat.Dense) *mat.Dense {
	// exponential
	U.Apply(func(_, _ int, v float64) float64 {
		return 2*math.Exp(-1*math.Pow(v-c.Mu, 2)/(2*math.Pow(c.Sigma, 2))) - 1
	}, U)
	return U
}

func (c *Config) Update() {
	// assume small world
	// @potential = elementwise_convolution(@kernel, @world)
	U := convolve(c.A, c.Kernel)
	// @growth = growth_mapping(@potential, mu, sigma)
	G := c.GrowthMapping(U)
	// @new_world = clip(@world + dt * @growth, 0, 1)
	G.Scale(c.Dt, G)
	A := mat.DenseCopyOf(c.A)
	A.Add(A, G)
	A.Apply(func(_, _ int, v float64) float64 {
		return Clip(v, 0, 1)
	}, A)
	// return @new_world, @growth, @potential
	c.A = mat.DenseCopyOf(A)
}

func padMatrix(m *mat.Dense, padding int) *mat.Dense {
	// add zero-padding around a matrix
	h, w := m.Dims()
	nh := h + 2*padding
	nw := w + 2*padding
	// full of zeros
	padded := mat.NewDense(nh, nw, fill(nh*nw, 0))
	// copy matrix at the center
	for i := 0; i < h; i++ {
		for j := 0; j < h; j++ {
			padded.Set(i+padding, j+padding, m.At(i, j))
		}
	}
	return padded
}

// TODO redefine with matrix multiplication (should be more efficient)
func convolve(m, kernel *mat.Dense) *mat.Dense {
	// perform a convolution between a matrix and a kernel matrix
	p, _ := kernel.Dims()
	p = int((p - 1) / 2)
	h, w := m.Dims()
	// n will receive the new computed values and ref is a copy of n before it is altered
	n := padMatrix(m, p)
	ref := mat.DenseCopyOf(n)
	n.Apply(func(i, j int, _ float64) float64 {
		// first flip the kernel, no need here as it is symmetrical
		// do not compute for padded values
		if i < p || i >= h+p || j < p || j >= w+p {
			return 0
		} else {
			// take a submatrix of n, same size as kernel
			subm := mat.DenseCopyOf(ref.Slice(i-p, i+p+1, j-p, j+p+1))
			// multiply element wise with kernel
			subm.MulElem(subm, kernel)
			// sum all elements
			sum := floats.Sum(subm.RawMatrix().Data)
			return sum
		}
	},
		n)
	// remove padding
	return mat.DenseCopyOf(n.Slice(p, h+p, p, w+p))
}
