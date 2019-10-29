package main

import "math"

const (
	typeJ = iota
	typeB
	typeE
	typeK
	typeN
	typeR
	typeS
	typeT
)

func SelectThermoCouple(thermoCoupleType string) int {
	switch thermoCoupleType {
	case "B":
		return typeB
	case "E":
		return typeE
	case "J":
		return typeJ
	case "K":
		return typeK
	case "N":
		return typeN
	case "R":
		return typeR
	case "S":
		return typeS
	default:
		return typeT
	}
}

func ConvertMvToTemp(v float64, thermoCoupleTypeId int) float64 {

	kT2VExp := []float64{0.1185976, -1.183432e-4, 126.9686}
	var result = 0.0
	var index = -1
	var i = 0
	coefficients, breakPoints, eqnOrders := SetTcGroup(thermoCoupleTypeId)
	for i < len(breakPoints)-1 {
		if v >= breakPoints[i] && v <= breakPoints[i+1] {
			index = i
			break
		}
		i++
	}
	if index >= 0 {
		order := eqnOrders[index]
		var coef []float64
		i2 := 0
		for i2 < order {
			coef = append(coef, coefficients[i2][index])
			i2++
		}
		result = PolyCalc(v, coef, order-1)
		if thermoCoupleTypeId == 3 && index == 1 {
			result += kT2VExp[0] * math.Exp(kT2VExp[1]*(v-kT2VExp[2])*(v-kT2VExp[2]))
		}
	}
	return result
}

func PolyCalc(x float64, coef []float64, order int) float64 {
	var y = coef[order]
	var i = order - 1
	for i >= 0 {
		y = y*x + coef[i]
		i--
	}
	return y
}

func SetTcGroup(tct int) ([][]float64, []float64, []int) {
	bV2TRanges := []float64{0.291, 2.431, 13.82}
	bV2TOrders := []int{9, 9}
	bV2TCoef := [][]float64{{98.423321, 213.15071}, {699.715, 285.10504}, {-847.65304, -52.742887}, {1005.2644, 9.9160804}, {-833.45952, -1.2965303}, {455.08542, 0.1119587}, {-155.23037, -0.0060625199}, {29.88675, 1.8661696e-4}, {-2.474286, -2.4878585e-6}}

	eV2TRanges := []float64{-8.825, 0.0, 76.373}
	eV2TOrders := []int{9, 10}
	eV2TCoef := [][]float64{{0.0, 0.0}, {16.977288, 17.057035}, {-0.4351497, -0.23301759}, {-0.15859697, 0.0065435585}, {-0.092502871, -7.3562749e-5}, {-0.026084314, -1.7896001e-6}, {-0.0041360199, 8.4036165e-8}, {-3.403403e-4, -1.3735879e-9}, {-1.156489e-5, 1.0629823e-11}, {0.0, -3.2447087e-14}}

	jV2TRanges := []float64{-8.095, 0.0, 42.919, 69.553}
	jV2TOrders := []int{9, 8, 6}
	jV2TCoef := [][]float64{{0.0, 0.0, -3113.58187}, {19.528268, 19.78425, 300.543684}, {-1.2286185, -0.2001204, -9.9477323}, {-1.0752178, 0.01036969, 0.17027663}, {-0.59086933, -2.549687e-4, -0.00143033468}, {-0.17256713, 3.585153e-6, 4.73886084e-6}, {-0.028131513, -5.344285e-8, 0.0}, {-0.002396337, 5.09989e-10, 0.0}, {-8.3823321e-5, 0.0, 0.0}}

	kV2TRanges := []float64{-5.891, 0.0, 20.644, 54.886}
	kV2TOrders := []int{9, 10, 7}
	kV2TCoef := [][]float64{{0.0, 0.0, -131.8058}, {25.173462, 25.08355, 48.30222}, {-1.1662878, 0.07860106, -1.646031}, {-1.0833638, -0.2503131, 0.05464731}, {-0.8977354, 0.0831527, -9.650715e-4}, {-0.37342377, -0.01228034, 8.802193e-6}, {-0.086632643, 9.804036e-4, -3.11081e-8}, {-0.010450598, -4.41303e-5, 0.0}, {-5.1920577e-4, 1.057734e-6, 0.0}, {0.0, -1.052755e-8, 0.0}}

	nV2TRanges := []float64{-3.99, 0.0, 20.613, 47.513}
	nV2TOrders := []int{10, 8, 6}
	nV2TCoef := [][]float64{{0.0, 0.0, 19.72485}, {38.436847, 38.6896, 33.00943}, {1.1010485, -1.08267, -0.3915159}, {5.2229312, 0.0470205, 0.009855391}, {7.2060525, -2.12169e-6, -1.274371e-4}, {5.8488586, -1.17272e-4, 7.767022e-7}, {2.7754916, 5.3928e-6, 0.0}, {0.77075166, -7.98156e-8, 0.0}, {0.11582665, 0.0, 0.0}, {0.0073138868, 0.0, 0.0}}

	rV2TRanges := []float64{-0.226, 1.923, 11.361, 19.739, 21.103}
	rV2TOrders := []int{11, 10, 6, 5}
	rV2TCoef := [][]float64{{0.0, 13.34584505, -81.99599416, 34061.77836}, {188.9138, 147.2644573, 155.3962042, -7023.729171}, {-93.83529, -18.44024844, -8.342197663, 558.2903813}, {130.68619, 4.031129726, 0.4279433549, -19.52394635}, {-227.0358, -0.624942836, -0.0119157791, 0.2560740231}, {351.45659, 0.06468412046, 1.492290091e-4, 0.0}, {-389.539, -0.004458750426, 0.0, 0.0}, {282.39471, 1.994710149e-4, 0.0, 0.0}, {-126.07281, -5.31340179e-6, 0.0, 0.0}, {31.353611, 6.481976217e-8, 0.0, 0.0}, {-3.3187769, 0.0, 0.0, 0.0}}

	sV2TRanges := []float64{-0.235, 1.874, 10.332, 17.536, 18.693}
	sV2TOrders := []int{10, 10, 6, 5}
	sV2TCoef := [][]float64{{0.0, 12.91507177, -80.87801117, 53338.75126}, {184.94946, 146.6298863, 162.1573104, -12358.92298}, {-80.0504062, -15.34713402, -8.536869453, 1092.657613}, {102.23743, 3.145945973, 0.4719686976, -42.65693686}, {-152.248592, -0.4163257839, -0.01441693666, 0.624720542}, {188.821343, 0.03187963771, 2.08161889e-4, 0.0}, {-159.085941, -0.0012916375, 0.0, 0.0}, {82.302788, 2.183475087e-5, 0.0, 0.0}, {-23.4181944, -1.447379511e-7, 0.0, 0.0}, {2.7978626, 8.211272125e-9, 0.0, 0.0}}

	tV2TRanges := []float64{-5.603, 0.0, 20.872}
	tV2TOrders := []int{8, 7}
	tV2TCoef := [][]float64{{0.0, 0.0}, {25.949192, 25.928}, {-0.21316967, -0.7602961}, {0.79018692, 0.04637791}, {0.42527777, -0.002165394}, {0.13304473, 6.048144e-5}, {0.020241446, -7.293422e-7}, {0.0012668171, 0.0}}

	switch tct {
	case 1:
		return bV2TCoef, bV2TRanges, bV2TOrders
	case 2:
		return eV2TCoef, eV2TRanges, eV2TOrders
	case 0:
		return jV2TCoef, jV2TRanges, jV2TOrders
	case 3:
		return kV2TCoef, kV2TRanges, kV2TOrders
	case 4:
		return nV2TCoef, nV2TRanges, nV2TOrders
	case 5:
		return rV2TCoef, rV2TRanges, rV2TOrders
	case 6:
		return sV2TCoef, sV2TRanges, sV2TOrders
	case 7:
		return tV2TCoef, tV2TRanges, tV2TOrders
	default:
		return jV2TCoef, jV2TRanges, jV2TOrders
	}
}
