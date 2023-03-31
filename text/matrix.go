package text

// LinearMatrix - used to perform spacial calculations for pdf object.  Uses linear algebra to
// determine movements and lengths of objects on a 2D plane.  And you thought you would never need
// to use algebra again. :-)  We only need a 2 x 3 matrix to represent 2D graphics, but multiplication
// requires the rows and columns have the same number of elements.  So the third column with 0 0 1 is
// added to support the calculations.
//   a  b  0
//   c  d  0
//   e  f  1
//
//   a = ScaleX = row 0, col 0
//   b = ShearX = row 0, col 1
//   c = ShearY = row 1, col 0
//   d = ScaleY = row 1, col 1
//   e = OffsetX = row 2, col 0
//   f = OffsetY = row 2, col 1
type LinearMatrix struct {
	gridRowCol [3][3]float64
}

// NewDefaultMatrix - create a pointer to a new matrix with default values
func NewDefaultMatrix() *LinearMatrix {
	m := LinearMatrix{}
	m.gridRowCol[0] = [3]float64{1.0, 0.0, 0.0}
	m.gridRowCol[1] = [3]float64{0.0, 1.0, 0.0}
	m.gridRowCol[2] = [3]float64{0.0, 0.0, 1.0}
	return &m
}

// Copy - create a pointer to a new matrix with the same values
func (l *LinearMatrix) Copy() *LinearMatrix {
	m := LinearMatrix{}
	for row := 0; row < 3; row++ {
		for col := 0; col < 3; col++ {
			m.gridRowCol[row][col] = l.gridRowCol[row][col]
		}
	}
	return &m
}

// Product - not a full 3x3 product since col 3 doesn't change. It's just there to aid in the math
func (l *LinearMatrix) Product(o *LinearMatrix) *LinearMatrix {
	p := LinearMatrix{}
	for i := 0; i < 3; i++ {
		for j := 0; j < 2; j++ {
			p.gridRowCol[i][j] = l.gridRowCol[i][j] * o.gridRowCol[j][i]
		}
	}
	return &p
}

func (l *LinearMatrix) Translate(offsetX float64, offsetY float64) {
	newX := l.GetScaleX()*offsetX + l.GetShearY()*offsetY + l.GetOffsetX()
	newY := l.GetShearX()*offsetX + l.GetScaleY()*offsetY + l.GetOffsetY()
	l.SetOffsetX(newX)
	l.SetOffsetY(newY)
}

func (l *LinearMatrix) GetScaleX() float64 {
	return l.gridRowCol[0][0]
}

func (l *LinearMatrix) SetScaleX(value float64) {
	l.gridRowCol[0][0] = value
}

func (l *LinearMatrix) GetShearX() float64 {
	return l.gridRowCol[0][1]
}

func (l *LinearMatrix) SetShearX(value float64) {
	l.gridRowCol[0][1] = value
}

func (l *LinearMatrix) GetShearY() float64 {
	return l.gridRowCol[1][0]
}

func (l *LinearMatrix) SetShearY(value float64) {
	l.gridRowCol[1][0] = value
}

func (l *LinearMatrix) GetScaleY() float64 {
	return l.gridRowCol[1][1]
}

func (l *LinearMatrix) SetScaleY(value float64) {
	l.gridRowCol[1][1] = value
}

func (l *LinearMatrix) GetOffsetX() float64 {
	return l.gridRowCol[2][0]
}

func (l *LinearMatrix) SetOffsetX(value float64) {
	l.gridRowCol[2][0] = value
}

func (l *LinearMatrix) GetOffsetY() float64 {
	return l.gridRowCol[2][1]
}

func (l *LinearMatrix) SetOffsetY(value float64) {
	l.gridRowCol[2][1] = value
}

// Set function handles operands from a Tm operator to set all aspects of the matrix.  Commonly found
// at the beginning of each text block with relative movements that follow.
func (l *LinearMatrix) Set(a float64, b float64, c float64, d float64, e float64, f float64) {
	l.gridRowCol[0][0] = a
	l.gridRowCol[0][1] = b
	l.gridRowCol[1][0] = c
	l.gridRowCol[1][1] = d
	l.gridRowCol[2][0] = e
	l.gridRowCol[2][1] = f
}

// Move supports a simple X/Y movement without so much math
func (l *LinearMatrix) Move(x float64, y float64) {
	l.SetOffsetX(l.GetOffsetX() + x)
	l.SetOffsetY(l.GetOffsetY() + y)
}
