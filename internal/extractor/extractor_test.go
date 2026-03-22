package extractor

import (
	"context"
	"image"
	"image/color"
	"path/filepath"
	"testing"
)

const testImage = "../../testdata/1-unsolved.png"

func TestDecodeImage(t *testing.T) {
	img, err := DecodeImage(testImage)
	if err != nil {
		t.Fatalf("DecodeImage: %v", err)
	}
	bounds := img.Bounds()
	if bounds.Dx() == 0 || bounds.Dy() == 0 {
		t.Fatal("decoded image has zero dimensions")
	}
	t.Logf("image size: %dx%d", bounds.Dx(), bounds.Dy())
}

func TestGrayscale(t *testing.T) {
	// Create a small color test image.
	src := image.NewRGBA(image.Rect(0, 0, 4, 4))
	src.Set(0, 0, color.RGBA{R: 255, G: 0, B: 0, A: 255})
	src.Set(1, 0, color.RGBA{R: 0, G: 255, B: 0, A: 255})
	src.Set(2, 0, color.RGBA{R: 0, G: 0, B: 255, A: 255})
	src.Set(3, 0, color.RGBA{R: 255, G: 255, B: 255, A: 255})

	gray := Grayscale(src)

	if gray.Bounds() != src.Bounds() {
		t.Fatalf("bounds mismatch: got %v, want %v", gray.Bounds(), src.Bounds())
	}

	// White pixel should remain white (255).
	if g := gray.GrayAt(3, 0).Y; g != 255 {
		t.Errorf("white pixel: got %d, want 255", g)
	}
	// Red pixel should have a grayscale value less than 255.
	if g := gray.GrayAt(0, 0).Y; g == 0 || g == 255 {
		t.Errorf("red pixel: got unexpected grayscale %d", g)
	}
}

func TestThreshold(t *testing.T) {
	gray := image.NewGray(image.Rect(0, 0, 4, 1))
	gray.SetGray(0, 0, color.Gray{Y: 0})   // black (0)
	gray.SetGray(1, 0, color.Gray{Y: 100}) // below 128
	gray.SetGray(2, 0, color.Gray{Y: 200}) // over 128
	gray.SetGray(3, 0, color.Gray{Y: 255}) // white (255)

	bin := Threshold(gray, 128)

	tests := []struct {
		x    int
		want uint8
	}{
		{0, 0},
		{1, 0},
		{2, 255},
		{3, 255},
	}
	for _, tc := range tests {
		got := bin.GrayAt(tc.x, 0).Y
		if got != tc.want {
			t.Errorf("pixel %d: got %d, want %d", tc.x, got, tc.want)
		}
	}
}

func TestFindGridBounds(t *testing.T) {
	img, err := DecodeImage(testImage)
	if err != nil {
		t.Fatalf("DecodeImage: %v", err)
	}

	gray := Grayscale(img)
	bin := Threshold(gray, 200)

	bounds, err := FindGridBounds(bin)
	if err != nil {
		t.Fatalf("FindGridBounds: %v", err)
	}

	imgW := img.Bounds().Dx()
	gridW := bounds.Dx()
	gridH := bounds.Dy()

	t.Logf("image width: %d, grid bounds: %v (%dx%d)", imgW, bounds, gridW, gridH)

	// Grid should take up at least 80% of the image width.
	if float64(gridW) < float64(imgW)*0.80 {
		t.Errorf("grid width %d is less than 80%% of image width %d", gridW, imgW)
	}

	// Grid should be roughly square.
	ratio := float64(gridW) / float64(gridH)
	if ratio < 0.8 || ratio > 1.2 {
		t.Errorf("grid aspect ratio %.2f is not roughly square", ratio)
	}
}

func TestSplitCells(t *testing.T) {
	// Create a 90x90 test image so cells divide evenly.
	src := image.NewGray(image.Rect(0, 0, 90, 90))
	bounds := src.Bounds()
	cells := SplitCells(src, bounds)

	for row := 0; row < 9; row++ {
		for col := 0; col < 9; col++ {
			cell := cells[row][col]
			if cell == nil {
				t.Errorf("cell [%d][%d] is nil", row, col)
				continue
			}
			cb := cell.Bounds()
			if cb.Dx() != 10 || cb.Dy() != 10 {
				t.Errorf("cell [%d][%d] size: %dx%d, want 10x10", row, col, cb.Dx(), cb.Dy())
			}
		}
	}
}

func TestExtractGrid(t *testing.T) {
	absPath, err := filepath.Abs(testImage)
	if err != nil {
		t.Fatalf("filepath.Abs: %v", err)
	}

	cells, err := ExtractGrid(context.Background(), absPath)
	if err != nil {
		t.Fatalf("ExtractGrid: %v", err)
	}

	for row := 0; row < 9; row++ {
		for col := 0; col < 9; col++ {
			cell := cells[row][col]
			if cell == nil {
				t.Errorf("cell [%d][%d] is nil", row, col)
				continue
			}
			cb := cell.Bounds()
			if cb.Dx() == 0 || cb.Dy() == 0 {
				t.Errorf("cell [%d][%d] has zero dimensions", row, col)
			}
		}
	}

	// All cells should be roughly the same size.
	refW := cells[0][0].Bounds().Dx()
	refH := cells[0][0].Bounds().Dy()
	for row := 0; row < 9; row++ {
		for col := 0; col < 9; col++ {
			cb := cells[row][col].Bounds()
			dw := abs(cb.Dx() - refW)
			dh := abs(cb.Dy() - refH)
			if dw > 2 || dh > 2 {
				t.Errorf("cell [%d][%d] size %dx%d differs from reference %dx%d", row, col, cb.Dx(), cb.Dy(), refW, refH)
			}
		}
	}

	t.Logf("cell size: ~%dx%d", refW, refH)
}

func TestExtract(t *testing.T) {
	tests := []struct {
		name     string
		image    string
		expected [9][9]int
	}{
		{
			name:  "1-unsolved.png",
			image: "../../testdata/1-unsolved.png",
			expected: [9][9]int{
				{3, 2, 0, 0, 0, 0, 1, 0, 0},
				{0, 0, 7, 0, 0, 0, 0, 0, 0},
				{5, 0, 0, 0, 7, 9, 0, 0, 3},
				{0, 0, 0, 0, 0, 0, 0, 0, 0},
				{4, 0, 0, 0, 0, 0, 5, 6, 2},
				{0, 0, 0, 6, 8, 0, 4, 1, 0},
				{0, 0, 0, 0, 0, 8, 0, 0, 0},
				{0, 8, 0, 9, 3, 0, 0, 0, 0},
				{2, 1, 9, 0, 5, 6, 0, 0, 0},
			},
		},
		{
			name:  "1-solved.png",
			image: "../../testdata/1-solved.png",
			expected: [9][9]int{
				{3, 2, 4, 8, 6, 5, 1, 7, 9},
				{8, 9, 7, 3, 2, 1, 6, 4, 5},
				{5, 6, 1, 4, 7, 9, 2, 8, 3},
				{1, 7, 6, 5, 4, 2, 3, 9, 8},
				{4, 3, 8, 1, 9, 7, 5, 6, 2},
				{9, 5, 2, 6, 8, 3, 4, 1, 7},
				{7, 4, 3, 2, 1, 8, 9, 5, 6},
				{6, 8, 5, 9, 3, 4, 7, 2, 1},
				{2, 1, 9, 7, 5, 6, 8, 3, 4},
			},
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			grid, err := Extract(context.Background(), tc.image)
			if err != nil {
				t.Fatalf("Extract: %v", err)
			}

			mismatches := 0
			for row := 0; row < 9; row++ {
				for col := 0; col < 9; col++ {
					if grid[row][col] != tc.expected[row][col] {
						t.Errorf("cell [%d][%d]: got %d, want %d", row, col, grid[row][col], tc.expected[row][col])
						mismatches++
					}
				}
			}

			t.Logf("extracted grid:")
			for row := 0; row < 9; row++ {
				t.Logf("  %v", grid[row])
			}

			if mismatches > 0 {
				t.Errorf("%d mismatches out of 81 cells", mismatches)
			}
		})
	}
}

func abs(x int) int {
	if x < 0 {
		return -x
	}
	return x
}
