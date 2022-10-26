package random

import (
	"ethattacksim/interfaces"
	"golang.org/x/exp/rand"
	"gonum.org/v1/gonum/stat/distuv"
	"log"
)

// add new distributions here
var normal *distuv.Normal
var uniform *distuv.Uniform
var normalCount int
var uniformCount int

func Normal() float64 {
	normalCount++
	return normal.Rand()
}

func Uniform() float64 {
	uniformCount++
	return uniform.Rand()
}

func PrintCount() {
	log.Printf("random number generators call count (indicates determinism) -> normal: %v, uniform: %v", normalCount, uniformCount)
}

func Initialize(seed uint64) {
	normalCount = 0
	uniformCount = 0
	// init normal dist rand num gen
	var normalSource rand.Source = rand.NewSource(seed)
	normal = &distuv.Normal{Mu: 0, Sigma: 1, Src: normalSource}

	// init uniform dist rand num gen
	var uniformSource rand.Source = rand.NewSource(seed)
	uniform = &distuv.Uniform{Min: 0, Max: 1, Src: uniformSource}

	// init new distributions here
}

func GetDist(distName string, params []float64, source rand.Source) interfaces.IRNG {
	switch distName {
	case "beta":
		return &distuv.Beta{Alpha: params[0], Beta: params[1], Src: source}
	case "invgamma":
		return &distuv.InverseGamma{Alpha: params[0], Beta: params[1], Src: source}
	case "norm":
		return &distuv.Normal{Mu: params[0], Sigma: params[1], Src: source}
	case "gamma":
		return &distuv.Gamma{Alpha: params[0], Beta: params[1], Src: source}
	case "lognorm":
		return &distuv.LogNormal{Mu: params[0], Sigma: params[1], Src: source}
	case "chisquare":
		return &distuv.ChiSquared{K: params[0], Src: source}
	case "exp":
		return &distuv.Exponential{Rate: params[0], Src: source}
	case "F":
		return &distuv.F{D1: params[0], D2: params[1], Src: source}
	case "laplace":
		return &distuv.Laplace{Mu: params[0], Scale: params[1], Src: source}
	case "pareto":
		return &distuv.Pareto{Xm: params[0], Alpha: params[1], Src: source}
	case "uniform":
		return &distuv.Uniform{Min: params[0], Max: params[1], Src: source}
	case "weibull":
		return &distuv.Weibull{K: params[0], Lambda: params[1], Src: source}
	default:
		log.Panic("distribution " + distName + " not found")
		return nil
	}
}
