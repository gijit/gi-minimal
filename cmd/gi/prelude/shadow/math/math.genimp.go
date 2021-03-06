package shadow_math

import "math"

var Pkg = make(map[string]interface{})
var Ctor = make(map[string]interface{})

func init() {
	Pkg["Abs"] = math.Abs
	Pkg["Acos"] = math.Acos
	Pkg["Acosh"] = math.Acosh
	Pkg["Asin"] = math.Asin
	Pkg["Asinh"] = math.Asinh
	Pkg["Atan"] = math.Atan
	Pkg["Atan2"] = math.Atan2
	Pkg["Atanh"] = math.Atanh
	Pkg["Cbrt"] = math.Cbrt
	Pkg["Ceil"] = math.Ceil
	Pkg["Copysign"] = math.Copysign
	Pkg["Cos"] = math.Cos
	Pkg["Cosh"] = math.Cosh
	Pkg["Dim"] = math.Dim
	Pkg["E"] = math.E
	Pkg["Erf"] = math.Erf
	Pkg["Erfc"] = math.Erfc
	Pkg["Erfcinv"] = math.Erfcinv
	Pkg["Erfinv"] = math.Erfinv
	Pkg["Exp"] = math.Exp
	Pkg["Exp2"] = math.Exp2
	Pkg["Expm1"] = math.Expm1
	Pkg["Float32bits"] = math.Float32bits
	Pkg["Float32frombits"] = math.Float32frombits
	Pkg["Float64bits"] = math.Float64bits
	Pkg["Float64frombits"] = math.Float64frombits
	Pkg["Floor"] = math.Floor
	Pkg["Frexp"] = math.Frexp
	Pkg["Gamma"] = math.Gamma
	Pkg["Hypot"] = math.Hypot
	Pkg["Ilogb"] = math.Ilogb
	Pkg["Inf"] = math.Inf
	Pkg["IsInf"] = math.IsInf
	Pkg["IsNaN"] = math.IsNaN
	Pkg["J0"] = math.J0
	Pkg["J1"] = math.J1
	Pkg["Jn"] = math.Jn
	Pkg["Ldexp"] = math.Ldexp
	Pkg["Lgamma"] = math.Lgamma
	Pkg["Ln10"] = math.Ln10
	Pkg["Ln2"] = math.Ln2
	Pkg["Log"] = math.Log
	Pkg["Log10"] = math.Log10
	Pkg["Log10E"] = math.Log10E
	Pkg["Log1p"] = math.Log1p
	Pkg["Log2"] = math.Log2
	Pkg["Log2E"] = math.Log2E
	Pkg["Logb"] = math.Logb
	Pkg["Max"] = math.Max
	Pkg["MaxFloat32"] = math.MaxFloat32
	Pkg["MaxFloat64"] = math.MaxFloat64
	Pkg["MaxInt16"] = math.MaxInt16
	Pkg["MaxInt32"] = math.MaxInt32
	Pkg["MaxInt64"] = math.MaxInt64
	Pkg["MaxInt8"] = math.MaxInt8
	Pkg["MaxUint16"] = math.MaxUint16
	Pkg["MaxUint32"] = math.MaxUint32
	Pkg["MaxUint64"] = uint64(math.MaxUint64)
	Pkg["MaxUint8"] = math.MaxUint8
	Pkg["Min"] = math.Min
	Pkg["MinInt16"] = math.MinInt16
	Pkg["MinInt32"] = math.MinInt32
	Pkg["MinInt64"] = math.MinInt64
	Pkg["MinInt8"] = math.MinInt8
	Pkg["Mod"] = math.Mod
	Pkg["Modf"] = math.Modf
	Pkg["NaN"] = math.NaN
	Pkg["Nextafter"] = math.Nextafter
	Pkg["Nextafter32"] = math.Nextafter32
	Pkg["Phi"] = math.Phi
	Pkg["Pi"] = math.Pi
	Pkg["Pow"] = math.Pow
	Pkg["Pow10"] = math.Pow10
	Pkg["Remainder"] = math.Remainder
	Pkg["Round"] = math.Round
	Pkg["RoundToEven"] = math.RoundToEven
	Pkg["Signbit"] = math.Signbit
	Pkg["Sin"] = math.Sin
	Pkg["Sincos"] = math.Sincos
	Pkg["Sinh"] = math.Sinh
	Pkg["SmallestNonzeroFloat32"] = math.SmallestNonzeroFloat32
	Pkg["SmallestNonzeroFloat64"] = math.SmallestNonzeroFloat64
	Pkg["Sqrt"] = math.Sqrt
	Pkg["Sqrt2"] = math.Sqrt2
	Pkg["SqrtE"] = math.SqrtE
	Pkg["SqrtPhi"] = math.SqrtPhi
	Pkg["SqrtPi"] = math.SqrtPi
	Pkg["Tan"] = math.Tan
	Pkg["Tanh"] = math.Tanh
	Pkg["Trunc"] = math.Trunc
	Pkg["Y0"] = math.Y0
	Pkg["Y1"] = math.Y1
	Pkg["Yn"] = math.Yn

}

func InitLua() string {
	return `
__type__.math ={};

`
}
