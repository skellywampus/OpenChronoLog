package main

import (
    "image"
    "image/color"
    "image/draw"
    "image/png"
    "math"
    "os"
    "path/filepath"
)

func main() {
    const size = 256
    img := image.NewRGBA(image.Rect(0, 0, size, size))

    // Background color
    bg := color.RGBA{0x1E, 0x88, 0xE5, 0xFF} // blue-ish
    draw.Draw(img, img.Bounds(), &image.Uniform{bg}, image.Point{}, draw.Src)

    cx, cy := size/2, size/2
    r := size/2 - 16
    white := color.RGBA{0xFF, 0xFF, 0xFF, 0xFF}

    // Clock circle (stroke)
    stroke := 6
    for y := 0; y < size; y++ {
        for x := 0; x < size; x++ {
            dx := float64(x - cx)
            dy := float64(y - cy)
            d := math.Hypot(dx, dy)
            if d > float64(r-stroke) && d < float64(r+stroke) {
                img.Set(x, y, white)
            }
        }
    }

    // Draw clock hands
    drawHand := func(angle float64, length int, thickness int) {
        for t := -thickness; t <= thickness; t++ {
            for l := 0; l < length; l++ {
                x := cx + int(float64(l)*math.Cos(angle)) + t
                y := cy + int(float64(l)*math.Sin(angle))
                if image.Pt(x, y).In(img.Bounds()) {
                    img.Set(x, y, white)
                }
            }
        }
    }
    toRad := func(deg float64) float64 { return (deg-90) * math.Pi / 180 }
    hourLen := int(float64(r) * 0.55)
    minLen := int(float64(r) * 0.75)
    // 10:10 look
    drawHand(toRad(300), hourLen, 2) // hour hand
    drawHand(toRad(60), minLen, 1)   // minute hand

    // Center dot
    for y := cy - 3; y <= cy + 3; y++ {
        for x := cx - 3; x <= cx + 3; x++ {
            if image.Pt(x, y).In(img.Bounds()) {
                img.Set(x, y, white)
            }
        }
    }

    // Write to go-version/icon.png (two levels up from tools/genicon)
    out := filepath.Join("..", "..", "icon.png")
    if err := writePNG(out, img); err != nil {
        // Fallback to current directory
        _ = writePNG("icon.png", img)
    }
}

func writePNG(path string, img image.Image) error {
    f, err := os.Create(path)
    if err != nil {
        return err
    }
    defer f.Close()
    return png.Encode(f, img)
}
