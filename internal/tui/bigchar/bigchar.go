// Package bigchar renders Chinese characters as large block art using half-block characters.
package bigchar

import (
	"image"
	"image/color"
	"image/draw"
	"os"
	"strings"

	"golang.org/x/image/font"
	"golang.org/x/image/font/opentype"
	"golang.org/x/image/math/fixed"
)

var loadedFace font.Face

func init() {
	// Try to load a CJK font from common system locations
	fontPaths := []string{
		// macOS
		"/System/Library/Fonts/STHeiti Light.ttc",
		"/System/Library/Fonts/PingFang.ttc",
		"/System/Library/Fonts/Hiragino Sans GB.ttc",
		"/Library/Fonts/Arial Unicode.ttf",
		// Linux
		"/usr/share/fonts/opentype/noto/NotoSansCJK-Regular.ttc",
		"/usr/share/fonts/noto-cjk/NotoSansCJK-Regular.ttc",
		"/usr/share/fonts/truetype/noto/NotoSansCJK-Regular.ttc",
		"/usr/share/fonts/truetype/droid/DroidSansFallbackFull.ttf",
		// Windows
		"C:\\Windows\\Fonts\\msyh.ttc",
		"C:\\Windows\\Fonts\\simsun.ttc",
	}

	for _, path := range fontPaths {
		data, err := os.ReadFile(path)
		if err != nil {
			continue
		}

		// Try parsing as font collection first
		if coll, err := opentype.ParseCollection(data); err == nil && coll.NumFonts() > 0 {
			if fnt, err := coll.Font(0); err == nil {
				if face, err := opentype.NewFace(fnt, &opentype.FaceOptions{
					Size: 64,
					DPI:  72,
				}); err == nil {
					loadedFace = face
					return
				}
			}
		}

		// Try parsing as single font
		if fnt, err := opentype.Parse(data); err == nil {
			if face, err := opentype.NewFace(fnt, &opentype.FaceOptions{
				Size: 64,
				DPI:  72,
			}); err == nil {
				loadedFace = face
				return
			}
		}
	}
}

// RenderBlock renders a character using half-block characters (▀▄█)
// cols and rows define the output size in terminal cells
func RenderBlock(char string, cols, rows int) string {
	if char == "" || loadedFace == nil {
		return ""
	}

	// Get the character rune
	r := []rune(char)[0]

	// Get font metrics for sizing
	bounds, _, _ := loadedFace.GlyphBounds(r)
	glyphWidth := (bounds.Max.X - bounds.Min.X).Ceil()
	glyphHeight := (bounds.Max.Y - bounds.Min.Y).Ceil()

	// Add padding around the glyph
	padding := 4
	srcWidth := glyphWidth + padding*2
	srcHeight := glyphHeight + padding*2

	// Ensure minimum size
	if srcWidth < 64 {
		srcWidth = 64
	}
	if srcHeight < 64 {
		srcHeight = 64
	}

	// Create source image at font's natural size
	srcImg := image.NewGray(image.Rect(0, 0, srcWidth, srcHeight))
	draw.Draw(srcImg, srcImg.Bounds(), &image.Uniform{color.Black}, image.Point{}, draw.Src)

	// Calculate baseline position
	x := (srcWidth - glyphWidth) / 2
	y := srcHeight - padding - bounds.Max.Y.Ceil()

	// Draw the character
	d := &font.Drawer{
		Dst:  srcImg,
		Src:  image.White,
		Face: loadedFace,
		Dot:  fixed.P(x, y),
	}
	d.DrawString(char)

	// Scale down to target size (rows*2 because half-blocks)
	targetWidth := cols
	targetHeight := rows * 2

	scaledImg := scaleDown(srcImg, targetWidth, targetHeight)

	// Convert to half-block characters
	return imageToHalfBlocks(scaledImg, cols, rows)
}

// scaleDown scales a grayscale image using area averaging
func scaleDown(src *image.Gray, dstWidth, dstHeight int) *image.Gray {
	srcBounds := src.Bounds()
	srcWidth := srcBounds.Max.X
	srcHeight := srcBounds.Max.Y

	dst := image.NewGray(image.Rect(0, 0, dstWidth, dstHeight))

	xRatio := float64(srcWidth) / float64(dstWidth)
	yRatio := float64(srcHeight) / float64(dstHeight)

	for dy := 0; dy < dstHeight; dy++ {
		for dx := 0; dx < dstWidth; dx++ {
			// Calculate source region
			sx1 := int(float64(dx) * xRatio)
			sy1 := int(float64(dy) * yRatio)
			sx2 := int(float64(dx+1) * xRatio)
			sy2 := int(float64(dy+1) * yRatio)

			if sx2 > srcWidth {
				sx2 = srcWidth
			}
			if sy2 > srcHeight {
				sy2 = srcHeight
			}

			// Average the pixels in the source region
			var sum int
			count := 0
			for sy := sy1; sy < sy2; sy++ {
				for sx := sx1; sx < sx2; sx++ {
					sum += int(src.GrayAt(sx, sy).Y)
					count++
				}
			}

			if count > 0 {
				dst.SetGray(dx, dy, color.Gray{Y: uint8(sum / count)})
			}
		}
	}

	return dst
}

// imageToHalfBlocks converts a grayscale image to half-block art
func imageToHalfBlocks(img *image.Gray, cols, rows int) string {
	var result strings.Builder

	for row := 0; row < rows; row++ {
		for col := 0; col < cols; col++ {
			// Each character cell represents 2 vertical pixels
			topY := row * 2
			bottomY := row*2 + 1

			topBright := getPixelBrightness(img, col, topY)
			bottomBright := getPixelBrightness(img, col, bottomY)

			// Threshold for "on"
			threshold := uint8(40)

			topOn := topBright > threshold
			bottomOn := bottomBright > threshold

			if topOn && bottomOn {
				result.WriteRune('█')
			} else if topOn {
				result.WriteRune('▀')
			} else if bottomOn {
				result.WriteRune('▄')
			} else {
				result.WriteRune(' ')
			}
		}
		if row < rows-1 {
			result.WriteRune('\n')
		}
	}

	return result.String()
}

func getPixelBrightness(img *image.Gray, x, y int) uint8 {
	if x < 0 || y < 0 || x >= img.Bounds().Max.X || y >= img.Bounds().Max.Y {
		return 0
	}
	return img.GrayAt(x, y).Y
}

// IsAvailable returns true if a CJK font was found
func IsAvailable() bool {
	return loadedFace != nil
}

// cache for rendered characters
var cache = make(map[string]string)

// GetCached returns cached big character or renders new one
func GetCached(char string, cols, rows int) string {
	if !IsAvailable() {
		return ""
	}

	key := char + string(rune(cols)) + string(rune(rows))
	if cached, ok := cache[key]; ok {
		return cached
	}

	rendered := RenderBlock(char, cols, rows)
	cache[key] = rendered
	return rendered
}
