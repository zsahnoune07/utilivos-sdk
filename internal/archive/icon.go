package archive

import (
	"bytes"
	"fmt"
	"image"
	"image/color"
	stddraw "image/draw"
	"image/png"
	"math"
	"os"
	"path/filepath"
	"strings"

	"github.com/utilivo/utilivo-sdk/internal/manifest"
)

const optimizedIconSize = 512

func OptimizeManifestIcon(root string, m manifest.Manifest, entries []Entry) ([]Entry, error) {
	rel := normalizeBuildIgnoreLine(m.Icon)
	if rel == "" || !strings.EqualFold(filepath.Ext(rel), ".png") {
		return entries, nil
	}

	entryIndex := -1
	for i := range entries {
		if entries[i].RelPath == rel {
			entryIndex = i
			break
		}
	}
	if entryIndex == -1 {
		return entries, nil
	}

	iconBytes, ok, err := optimizePNGIcon(filepath.Join(root, filepath.FromSlash(rel)))
	if err != nil {
		return nil, fmt.Errorf("optimize manifest icon %s: %w", rel, err)
	}
	if !ok {
		return entries, nil
	}

	entries[entryIndex].Content = iconBytes
	entries[entryIndex].Size = int64(len(iconBytes))
	return entries, nil
}

func optimizePNGIcon(path string) ([]byte, bool, error) {
	sourceBytes, err := os.ReadFile(path)
	if err != nil {
		return nil, false, err
	}

	sourceImage, err := png.Decode(bytes.NewReader(sourceBytes))
	if err != nil {
		return nil, false, nil
	}

	canvas, err := normalizeIconCanvas(sourceImage)
	if err != nil {
		return nil, false, err
	}

	var out bytes.Buffer
	encoder := png.Encoder{CompressionLevel: png.BestCompression}
	if err := encoder.Encode(&out, canvas); err != nil {
		return nil, false, err
	}
	return out.Bytes(), true, nil
}

func normalizeIconCanvas(src image.Image) (*image.NRGBA, error) {
	bounds := src.Bounds()
	srcWidth := bounds.Dx()
	srcHeight := bounds.Dy()
	if srcWidth <= 0 || srcHeight <= 0 {
		return nil, fmt.Errorf("icon image must have positive dimensions")
	}

	scale := math.Min(
		float64(optimizedIconSize)/float64(srcWidth),
		float64(optimizedIconSize)/float64(srcHeight),
	)
	targetWidth := max(1, int(math.Round(float64(srcWidth)*scale)))
	targetHeight := max(1, int(math.Round(float64(srcHeight)*scale)))

	scaled := image.NewNRGBA(image.Rect(0, 0, targetWidth, targetHeight))
	resizeBilinear(scaled, src)

	canvas := image.NewNRGBA(image.Rect(0, 0, optimizedIconSize, optimizedIconSize))
	offset := image.Pt(
		(optimizedIconSize-targetWidth)/2,
		(optimizedIconSize-targetHeight)/2,
	)
	stddraw.Draw(
		canvas,
		image.Rectangle{Min: offset, Max: offset.Add(scaled.Bounds().Size())},
		scaled,
		image.Point{},
		stddraw.Src,
	)
	return canvas, nil
}

func resizeBilinear(dst *image.NRGBA, src image.Image) {
	dstBounds := dst.Bounds()
	srcBounds := src.Bounds()
	dstWidth := dstBounds.Dx()
	dstHeight := dstBounds.Dy()
	srcWidth := srcBounds.Dx()
	srcHeight := srcBounds.Dy()

	if dstWidth == 0 || dstHeight == 0 || srcWidth == 0 || srcHeight == 0 {
		return
	}

	if dstWidth == srcWidth && dstHeight == srcHeight {
		for y := 0; y < dstHeight; y++ {
			for x := 0; x < dstWidth; x++ {
				dst.Set(x, y, src.At(srcBounds.Min.X+x, srcBounds.Min.Y+y))
			}
		}
		return
	}

	scaleX := float64(srcWidth) / float64(dstWidth)
	scaleY := float64(srcHeight) / float64(dstHeight)

	for y := 0; y < dstHeight; y++ {
		srcY := (float64(y)+0.5)*scaleY - 0.5
		y0, y1, wy := interpolationWindow(srcY, srcHeight)
		for x := 0; x < dstWidth; x++ {
			srcX := (float64(x)+0.5)*scaleX - 0.5
			x0, x1, wx := interpolationWindow(srcX, srcWidth)

			c00 := rgbaFloats(src.At(srcBounds.Min.X+x0, srcBounds.Min.Y+y0))
			c10 := rgbaFloats(src.At(srcBounds.Min.X+x1, srcBounds.Min.Y+y0))
			c01 := rgbaFloats(src.At(srcBounds.Min.X+x0, srcBounds.Min.Y+y1))
			c11 := rgbaFloats(src.At(srcBounds.Min.X+x1, srcBounds.Min.Y+y1))

			top := mixChannels(c00, c10, wx)
			bottom := mixChannels(c01, c11, wx)
			pixel := mixChannels(top, bottom, wy)
			dst.SetNRGBA(x, y, toNRGBA(pixel))
		}
	}
}

type rgbaChannel struct {
	r float64
	g float64
	b float64
	a float64
}

func interpolationWindow(position float64, upperBound int) (int, int, float64) {
	if upperBound <= 1 {
		return 0, 0, 0
	}

	base := int(math.Floor(position))
	weight := position - float64(base)
	if base < 0 {
		return 0, 0, 0
	}
	if base >= upperBound-1 {
		last := upperBound - 1
		return last, last, 0
	}
	return base, base + 1, weight
}

func rgbaFloats(c color.Color) rgbaChannel {
	const maxChannel = 65535.0
	r, g, b, a := c.RGBA()
	return rgbaChannel{
		r: float64(r) / maxChannel,
		g: float64(g) / maxChannel,
		b: float64(b) / maxChannel,
		a: float64(a) / maxChannel,
	}
}

func mixChannels(a, b rgbaChannel, weight float64) rgbaChannel {
	return rgbaChannel{
		r: a.r + (b.r-a.r)*weight,
		g: a.g + (b.g-a.g)*weight,
		b: a.b + (b.b-a.b)*weight,
		a: a.a + (b.a-a.a)*weight,
	}
}

func toNRGBA(channel rgbaChannel) color.NRGBA {
	alpha := clamp01(channel.a)
	if alpha == 0 {
		return color.NRGBA{}
	}

	return color.NRGBA{
		R: toByte(channel.r / alpha),
		G: toByte(channel.g / alpha),
		B: toByte(channel.b / alpha),
		A: toByte(alpha),
	}
}

func toByte(value float64) uint8 {
	return uint8(math.Round(clamp01(value) * 255))
}

func clamp01(value float64) float64 {
	return math.Min(1, math.Max(0, value))
}
