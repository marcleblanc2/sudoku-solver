// Package extractor extracts a Sudoku grid from an input image (screenshot).
// Handles image processing to locate the grid and split it into 81 cell images.
package extractor

import (
	"bytes"
	"context"
	"fmt"
	"image"
	"image/color"
	_ "image/jpeg"
	"image/png"
	"os"
	"strings"

	"github.com/otiai10/gosseract/v2"
)

// DecodeImage reads an image file (PNG or JPEG) and returns the decoded image.
func DecodeImage(path string) (image.Image, error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, fmt.Errorf("open image: %w", err)
	}
	defer f.Close()

	img, _, err := image.Decode(f)
	if err != nil {
		return nil, fmt.Errorf("decode image: %w", err)
	}
	return img, nil
}

// Grayscale converts an image to 8-bit grayscale.
func Grayscale(img image.Image) *image.Gray {
	bounds := img.Bounds()
	gray := image.NewGray(bounds)
	for y := bounds.Min.Y; y < bounds.Max.Y; y++ {
		for x := bounds.Min.X; x < bounds.Max.X; x++ {
			gray.Set(x, y, color.GrayModel.Convert(img.At(x, y)))
		}
	}
	return gray
}

// Threshold converts a grayscale image to binary (black and white).
// Pixels with intensity <= threshold become black (0); others become white (255).
func Threshold(img *image.Gray, threshold uint8) *image.Gray {
	bounds := img.Bounds()
	bin := image.NewGray(bounds)
	for y := bounds.Min.Y; y < bounds.Max.Y; y++ {
		for x := bounds.Min.X; x < bounds.Max.X; x++ {
			if img.GrayAt(x, y).Y <= threshold {
				bin.SetGray(x, y, color.Gray{Y: 0})
			} else {
				bin.SetGray(x, y, color.Gray{Y: 255})
			}
		}
	}
	return bin
}

// FindGridBounds locates the Sudoku grid in a binary image by analysing
// horizontal and vertical projections of black pixels. It returns the
// bounding rectangle of the grid.
//
// Grid lines are identified as rows/columns where black pixels span nearly
// the full image width/height (>90% density). The grid is bounded by the
// first and last cluster of such lines, but we stop when a large continuous
// dark region is found (e.g. a keypad below the grid).
func FindGridBounds(img *image.Gray) (image.Rectangle, error) {
	bounds := img.Bounds()
	w := bounds.Dx()

	// Horizontal projection: count black pixels per row.
	hProj := make([]int, bounds.Dy())
	for y := bounds.Min.Y; y < bounds.Max.Y; y++ {
		for x := bounds.Min.X; x < bounds.Max.X; x++ {
			if img.GrayAt(x, y).Y == 0 {
				hProj[y-bounds.Min.Y]++
			}
		}
	}

	// Identify grid-line rows: rows where >90% of pixels are black.
	// These are the thin/thick lines that span the full grid width.
	hThresh := int(float64(w) * 0.90)
	gridLineRows := findGridLinePositions(hProj, hThresh)
	if len(gridLineRows) == 0 {
		return image.Rectangle{}, fmt.Errorf("could not detect horizontal grid lines")
	}

	firstRow := gridLineRows[0]
	lastRow := gridLineRows[len(gridLineRows)-1]

	// Vertical projection: count black pixels per column, but only within
	// the detected row range to avoid interference from other UI elements.
	vProj := make([]int, w)
	for x := bounds.Min.X; x < bounds.Max.X; x++ {
		for y := bounds.Min.Y + firstRow; y <= bounds.Min.Y+lastRow; y++ {
			if img.GrayAt(x, y).Y == 0 {
				vProj[x-bounds.Min.X]++
			}
		}
	}

	// For columns, grid lines should have >30% density within the grid's
	// vertical span (they are thin lines, a few pixels wide, spanning
	// the grid height).
	gridH := lastRow - firstRow + 1
	vThresh := int(float64(gridH) * 0.30)
	firstCol, lastCol := -1, -1
	for i, count := range vProj {
		if count >= vThresh {
			if firstCol == -1 {
				firstCol = i
			}
			lastCol = i
		}
	}

	if firstCol == -1 {
		return image.Rectangle{}, fmt.Errorf("could not detect vertical grid lines")
	}

	gridW := lastCol - firstCol + 1

	// Validate roughly square aspect ratio.
	ratio := float64(gridW) / float64(gridH)
	if ratio < 0.8 || ratio > 1.2 {
		return image.Rectangle{}, fmt.Errorf("detected region is not square (aspect ratio %.2f)", ratio)
	}

	rect := image.Rect(
		bounds.Min.X+firstCol,
		bounds.Min.Y+firstRow,
		bounds.Min.X+lastCol+1,
		bounds.Min.Y+lastRow+1,
	)
	return rect, nil
}

// findGridLinePositions returns the row indices that are grid lines, filtering
// out large continuous dark regions (like a keypad) that aren't part of the
// Sudoku grid. A Sudoku grid has 10 horizontal lines; we identify narrow runs
// of dark rows as grid lines, and for wider runs (where the bottom border
// merges with other dark UI), we include only the first few rows.
func findGridLinePositions(proj []int, threshold int) []int {
	type run struct{ start, end int }
	var runs []run
	i := 0
	for i < len(proj) {
		if proj[i] >= threshold {
			start := i
			for i < len(proj) && proj[i] >= threshold {
				i++
			}
			runs = append(runs, run{start, i - 1})
		} else {
			i++
		}
	}

	if len(runs) == 0 {
		return nil
	}

	// Determine the typical width of a thick grid line from narrow runs.
	// Thick block dividers are ~7 pixels; thin cell dividers are ~3 pixels.
	maxNarrowWidth := 15
	maxThick := 0
	for _, r := range runs {
		w := r.end - r.start + 1
		if w <= maxNarrowWidth && w > maxThick {
			maxThick = w
		}
	}
	if maxThick == 0 {
		maxThick = 7
	}

	var gridLines []int
	for _, r := range runs {
		w := r.end - r.start + 1
		if w <= maxNarrowWidth {
			// Normal grid line — include all rows.
			for row := r.start; row <= r.end; row++ {
				gridLines = append(gridLines, row)
			}
		} else {
			// Wide run: the bottom border merging with other dark content.
			// Include only the first maxThick rows as the border, then stop
			// processing — anything beyond is not part of the grid.
			for row := r.start; row < r.start+maxThick && row <= r.end; row++ {
				gridLines = append(gridLines, row)
			}
			break
		}
	}
	return gridLines
}

// SplitCells crops the image to the grid bounds and divides it into a
// 9×9 array of cell images.
func SplitCells(img image.Image, bounds image.Rectangle) [9][9]image.Image {
	cellW := bounds.Dx() / 9
	cellH := bounds.Dy() / 9

	var cells [9][9]image.Image
	for row := 0; row < 9; row++ {
		for col := 0; col < 9; col++ {
			x0 := bounds.Min.X + col*cellW
			y0 := bounds.Min.Y + row*cellH
			x1 := x0 + cellW
			y1 := y0 + cellH
			// Last column/row absorbs any rounding remainder.
			if col == 8 {
				x1 = bounds.Max.X
			}
			if row == 8 {
				y1 = bounds.Max.Y
			}
			cells[row][col] = cropImage(img, image.Rect(x0, y0, x1, y1))
		}
	}
	return cells
}

// cropImage returns a sub-image for the given rectangle. If the source
// supports SubImage, it uses that directly; otherwise it copies pixels.
func cropImage(img image.Image, rect image.Rectangle) image.Image {
	type subImager interface {
		SubImage(r image.Rectangle) image.Image
	}
	if si, ok := img.(subImager); ok {
		return si.SubImage(rect)
	}

	dst := image.NewRGBA(image.Rect(0, 0, rect.Dx(), rect.Dy()))
	for y := rect.Min.Y; y < rect.Max.Y; y++ {
		for x := rect.Min.X; x < rect.Max.X; x++ {
			dst.Set(x-rect.Min.X, y-rect.Min.Y, img.At(x, y))
		}
	}
	return dst
}

// ExtractGrid reads a screenshot and returns the 9×9 array of cell images.
func ExtractGrid(ctx context.Context, path string) ([9][9]image.Image, error) {
	img, err := DecodeImage(path)
	if err != nil {
		return [9][9]image.Image{}, err
	}

	gray := Grayscale(img)
	bin := Threshold(gray, 200)

	bounds, err := FindGridBounds(bin)
	if err != nil {
		return [9][9]image.Image{}, fmt.Errorf("find grid bounds: %w", err)
	}

	cells := SplitCells(img, bounds)
	return cells, nil
}

// RecognizeDigits takes a 9x9 array of cell images and returns a 9x9 grid
// of digits (1-9, or 0 for empty cells).
func RecognizeDigits(cells [9][9]image.Image) ([9][9]int, error) {
	client := gosseract.NewClient()
	defer client.Close()

	if err := client.SetWhitelist("123456789"); err != nil {
		return [9][9]int{}, fmt.Errorf("set whitelist: %w", err)
	}
	if err := client.SetPageSegMode(gosseract.PSM_SINGLE_CHAR); err != nil {
		return [9][9]int{}, fmt.Errorf("set PSM: %w", err)
	}
	// Force LSTM-only engine for better single-digit accuracy.
	if err := client.SetVariable("tessedit_ocr_engine_mode", "1"); err != nil {
		return [9][9]int{}, fmt.Errorf("set OEM: %w", err)
	}

	var grid [9][9]int
	for row := 0; row < 9; row++ {
		for col := 0; col < 9; col++ {
			digit, err := recognizeCell(client, cells[row][col])
			if err != nil {
				return [9][9]int{}, fmt.Errorf("cell [%d][%d]: %w", row, col, err)
			}
			grid[row][col] = digit
		}
	}
	return grid, nil
}

// recognizeCell processes a single cell image and returns the digit (1-9)
// or 0 if the cell is empty.
func recognizeCell(client *gosseract.Client, cell image.Image) (int, error) {
	bounds := cell.Bounds()
	w := bounds.Dx()
	h := bounds.Dy()

	// Crop to inner ~62% to remove grid line borders.
	marginX := int(float64(w) * 0.19)
	marginY := int(float64(h) * 0.19)
	inner := image.Rect(
		bounds.Min.X+marginX,
		bounds.Min.Y+marginY,
		bounds.Max.X-marginX,
		bounds.Max.Y-marginY,
	)

	iw := inner.Dx()
	ih := inner.Dy()

	// Convert the inner region to a "min-channel" grayscale so that
	// coloured digits (e.g. teal user-filled values) become dark rather
	// than being washed out by standard luminance conversion.
	total := iw * ih
	minCh := make([]uint8, total)
	for y := inner.Min.Y; y < inner.Max.Y; y++ {
		for x := inner.Min.X; x < inner.Max.X; x++ {
			r, g, b, _ := cell.At(x, y).RGBA()
			// RGBA returns 16-bit values; shift to 8-bit.
			r8, g8, b8 := uint8(r>>8), uint8(g>>8), uint8(b>>8)
			m := r8
			if g8 < m {
				m = g8
			}
			if b8 < m {
				m = b8
			}
			minCh[(y-inner.Min.Y)*iw+(x-inner.Min.X)] = m
		}
	}

	// Compute the mean min-channel value for adaptive thresholding.
	var sum int64
	for _, v := range minCh {
		sum += int64(v)
	}
	mean := float64(sum) / float64(total)

	// Count dark pixels (below 90% of mean). Empty cells are nearly
	// uniform, so very few pixels deviate from the mean.
	darkCount := 0
	for _, v := range minCh {
		if float64(v) < mean*0.90 {
			darkCount++
		}
	}
	darkRatio := float64(darkCount) / float64(total)
	if darkRatio < 0.01 {
		return 0, nil
	}

	// Build a padded binary image: pixels darker than 90% of the mean
	// become black (digit ink), everything else becomes white.
	pad := iw / 3
	padded := image.NewGray(image.Rect(0, 0, iw+2*pad, ih+2*pad))
	for y := 0; y < ih+2*pad; y++ {
		for x := 0; x < iw+2*pad; x++ {
			padded.SetGray(x, y, color.Gray{Y: 255})
		}
	}

	thresh := mean * 0.90
	for y := 0; y < ih; y++ {
		for x := 0; x < iw; x++ {
			if float64(minCh[y*iw+x]) < thresh {
				padded.SetGray(x+pad, y+pad, color.Gray{Y: 0})
			}
		}
	}

	// Remove border-connected dark pixels (grid line residue) via flood fill.
	clearBorderBlobs(padded)

	// Dilate (thicken) strokes by 1 pixel to make thin digits more robust
	// for OCR recognition.
	padded = dilate(padded)

	// Encode as PNG for Tesseract.
	var buf bytes.Buffer
	if err := png.Encode(&buf, padded); err != nil {
		return 0, fmt.Errorf("encode png: %w", err)
	}

	if err := client.SetImageFromBytes(buf.Bytes()); err != nil {
		return 0, fmt.Errorf("set image: %w", err)
	}

	text, err := client.Text()
	if err != nil {
		return 0, fmt.Errorf("ocr: %w", err)
	}

	text = strings.TrimSpace(text)
	if len(text) == 1 && text[0] >= '1' && text[0] <= '9' {
		return int(text[0] - '0'), nil
	}
	return 0, nil
}

// dilate performs binary dilation: a white pixel becomes black if any of
// its 4 direct neighbours is black. This thickens digit strokes by 1px.
func dilate(img *image.Gray) *image.Gray {
	b := img.Bounds()
	dst := image.NewGray(b)
	for y := b.Min.Y; y < b.Max.Y; y++ {
		for x := b.Min.X; x < b.Max.X; x++ {
			if img.GrayAt(x, y).Y == 0 ||
				(x > b.Min.X && img.GrayAt(x-1, y).Y == 0) ||
				(x < b.Max.X-1 && img.GrayAt(x+1, y).Y == 0) ||
				(y > b.Min.Y && img.GrayAt(x, y-1).Y == 0) ||
				(y < b.Max.Y-1 && img.GrayAt(x, y+1).Y == 0) {
				dst.SetGray(x, y, color.Gray{Y: 0})
			} else {
				dst.SetGray(x, y, color.Gray{Y: 255})
			}
		}
	}
	return dst
}

// otsuThreshold computes the optimal binary threshold for a slice of pixel
// values using Otsu's method, which maximises inter-class variance.
func otsuThreshold(pixels []uint8) uint8 {
	var hist [256]int
	for _, v := range pixels {
		hist[v]++
	}
	total := len(pixels)

	var sumAll float64
	for i := 0; i < 256; i++ {
		sumAll += float64(i) * float64(hist[i])
	}

	var wB, wF int
	var sumB float64
	best := 0.0
	threshold := uint8(0)
	for t := 0; t < 256; t++ {
		wB += hist[t]
		if wB == 0 {
			continue
		}
		wF = total - wB
		if wF == 0 {
			break
		}
		sumB += float64(t) * float64(hist[t])
		mB := sumB / float64(wB)
		mF := (sumAll - sumB) / float64(wF)
		between := float64(wB) * float64(wF) * (mB - mF) * (mB - mF)
		if between > best {
			best = between
			threshold = uint8(t)
		}
	}
	return threshold
}

// clearBorderBlobs flood-fills from all dark (black) pixels on the image
// border, setting them and any connected dark pixels to white. This removes
// grid line residue that bleeds into the cropped cell region.
func clearBorderBlobs(img *image.Gray) {
	b := img.Bounds()
	w, h := b.Dx(), b.Dy()

	// Collect border seed pixels.
	type point struct{ x, y int }
	var seeds []point
	for x := b.Min.X; x < b.Max.X; x++ {
		if img.GrayAt(x, b.Min.Y).Y == 0 {
			seeds = append(seeds, point{x, b.Min.Y})
		}
		if img.GrayAt(x, b.Max.Y-1).Y == 0 {
			seeds = append(seeds, point{x, b.Max.Y - 1})
		}
	}
	for y := b.Min.Y; y < b.Max.Y; y++ {
		if img.GrayAt(b.Min.X, y).Y == 0 {
			seeds = append(seeds, point{b.Min.X, y})
		}
		if img.GrayAt(b.Max.X-1, y).Y == 0 {
			seeds = append(seeds, point{b.Max.X - 1, y})
		}
	}

	// BFS flood fill.
	visited := make([]bool, w*h)
	queue := seeds
	for len(queue) > 0 {
		p := queue[0]
		queue = queue[1:]
		idx := (p.y-b.Min.Y)*w + (p.x - b.Min.X)
		if idx < 0 || idx >= len(visited) || visited[idx] {
			continue
		}
		if img.GrayAt(p.x, p.y).Y != 0 {
			continue
		}
		visited[idx] = true
		img.SetGray(p.x, p.y, color.Gray{Y: 255})
		for _, d := range [][2]int{{-1, 0}, {1, 0}, {0, -1}, {0, 1}} {
			nx, ny := p.x+d[0], p.y+d[1]
			if nx >= b.Min.X && nx < b.Max.X && ny >= b.Min.Y && ny < b.Max.Y {
				queue = append(queue, point{nx, ny})
			}
		}
	}
}

// scaleUp scales a grayscale image by the given factor using nearest-neighbour
// interpolation to preserve sharp digit edges.
func scaleUp(img *image.Gray, factor int) *image.Gray {
	b := img.Bounds()
	dst := image.NewGray(image.Rect(0, 0, b.Dx()*factor, b.Dy()*factor))
	for y := b.Min.Y; y < b.Max.Y; y++ {
		for x := b.Min.X; x < b.Max.X; x++ {
			c := img.GrayAt(x, y)
			for dy := 0; dy < factor; dy++ {
				for dx := 0; dx < factor; dx++ {
					dst.SetGray((x-b.Min.X)*factor+dx, (y-b.Min.Y)*factor+dy, c)
				}
			}
		}
	}
	return dst
}

// Extract reads a screenshot and returns the 9x9 grid of digits.
func Extract(ctx context.Context, path string) ([9][9]int, error) {
	cells, err := ExtractGrid(ctx, path)
	if err != nil {
		return [9][9]int{}, err
	}
	return RecognizeDigits(cells)
}
