package utils

import (
	"math"
	"math/rand"
	"time"

	"github.com/mjibson/go-dsp/fft"
	"gonum.org/v1/gonum/floats"
	"gonum.org/v1/gonum/mat"
)

// for random generation
var seed = rand.NewSource(time.Now().UnixNano())
var r0 = rand.New(seed)

type Config struct {
	// matrices
	A, Kernel, G *mat.Dense
	KFFT         *mat.CDense
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

func randInt(min, max int) int {
	return r0.Intn(max-min) + min
}

func mod(a, b int) int {
	return (a%b + b) % b
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

func DenseToSlice(m *mat.Dense) [][]float64 {
	// convert a Dense matrix to a 2D slice
	var data [][]float64
	rawData := m.RawMatrix().Data
	r, c := m.Dims()
	for i := 0; i < r*c; i += c {
		data = append(data, rawData[i:i+c])
	}
	return data
}

func ComplexDenseToSlice(m *mat.CDense) [][]complex128 {
	// convert a complex Dense matrix to a complex 2D slice
	var data [][]complex128
	rawData := m.RawCMatrix().Data
	r, c := m.Dims()
	for i := 0; i < r*c; i += c {
		data = append(data, rawData[i:i+c])
	}
	return data
}

func ComplexSliceToDense(array [][]complex128) *mat.CDense {
	// convert complex 2D slice to a complex Dense matrix
	var data []complex128
	r := len(array)
	c := len(array[0])
	for _, e := range array {
		data = append(data, e...)
	}
	return mat.NewCDense(r, c, data)
}

func FFT(m *mat.Dense) *mat.CDense {
	// Fast Fourier Transform
	return ComplexSliceToDense(fft.FFT2Real(DenseToSlice(m)))
}

func IFFT(m *mat.CDense) *mat.CDense {
	// Inverse FFT
	return ComplexSliceToDense(fft.IFFT2(ComplexDenseToSlice(m)))
}

func FFTShift(m *mat.Dense, r, c int) *mat.Dense {
	shifted := mat.NewDense(r, c, nil)
	width, _ := m.Dims()
	R := int((width - 1) / 2)
	for i := -R; i <= R; i++ {
		for j := -R; j <= R; j++ {
			v := m.At(i+R, j+R)
			shifted.Set(mod(i, c), mod(j, c), v)
		}
	}
	return shifted
}

func RealPart(m *mat.CDense) *mat.Dense {
	r, c := m.Dims()
	realMatrix := mat.NewDense(r, c, nil)
	realMatrix.Apply(func(i, j int, _ float64) float64 {
		return real(m.At(i, j))
	}, realMatrix)
	return realMatrix
}

func ComplexMulElem(m1, m2 *mat.CDense) *mat.CDense {
	r, c := m1.Dims()
	result := mat.NewCDense(r, c, nil)
	// commented is the addition of concurrency wich doesn't seems to improve performances here
	//wg := sync.WaitGroup{}
	for i := 0; i < r; i++ {
		//wg.Add(1)
		//go func() {
		for j := 0; j < r; j++ {
			z1 := m1.At(i, j)
			z2 := m2.At(i, j)
			x1 := real(z1)
			y1 := imag(z1)
			x2 := real(z2)
			y2 := imag(z2)
			z := complex((x1*x2 - y1*y2), (x1*y2 + x2*y1))
			result.Set(i, j, z)
		}
		//wg.Done()
		//}()
		//wg.Wait()
	}
	return result
}

func (c *Config) InitState() {
	// define the initial state of A
	// for now, random rectangles
	h, w := c.A.Dims()
	// random number of rectagles
	for k := 0; k < randInt(15, 25); k++ {
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

func (c *Config) InitStateFull() {
	// define the initial state of A
	// for now, random rectangles
	h, w := c.A.Dims()
	// fill the rectangle to 1
	for i := 0; i < w; i++ {
		for j := 0; j < h; j++ {
			c.A.Set(i, j, r0.Float64())
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

func KernelCorePoly(r float64) float64 {
	var a float64 = 4
	value := math.Pow(4*r*(1-r), a)
	return value
}

func KernelCoreExp(r float64) float64 {
	var a float64 = 4
	value := math.Exp(a - a/(4*r*(1-r)))
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
		return c.Beta[int(math.Floor(v))] * KernelCoreExp(math.Mod(v, 1))
	}, K)
	// @kernel = @kernel_shell / sum(@kernel_shell)
	KS.Scale(1/floats.Sum(KS.RawMatrix().Data), KS)
	// @kernel_FFT = FFT_2D(@kernel)
	rows, cols := c.A.Dims()
	c.KFFT = FFT(FFTShift(KS, rows, cols))
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
	//start := time.Now()
	var U *mat.Dense
	// if size(@world) is small
	if false {
		// @potential = elementwise_convolution(@kernel, @world)
		U = convolve(c.A, c.Kernel)
	} else {
		// @world_FFT = FFT_2D(@world)
		AFFT := FFT(c.A)
		// @potential_FFT = elementwise_multiply(@kernel_FFT, @world_FFT)
		// @potential = FFT_shift(real_part(inverse_FFT_2D(@potential_FFT)))
		U = RealPart(IFFT(ComplexMulElem(c.KFFT, AFFT)))
	}
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
	//elapsed := time.Since(start)
	//fmt.Println("time elapsed:", elapsed)
}

func padMatrix(m *mat.Dense, padding int) *mat.Dense {
	// add zero-padding around a matrix
	h, w := m.Dims()
	nh := h + 2*padding
	nw := w + 2*padding
	// full of zeros
	padded := mat.NewDense(nh, nw, nil)
	// copy matrix at the center
	for i := 0; i < h; i++ {
		for j := 0; j < h; j++ {
			padded.Set(i+padding, j+padding, m.At(i, j))
		}
	}
	return padded
}

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
