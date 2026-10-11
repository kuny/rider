package main

import (
	"encoding/binary"
	"fmt"
	"image"
	_ "image/png"
	"math"
	"os"
)

const (
	sampleRate = 44100
)

func logisticMap(n int, initialX float64) float64 {
	x := initialX
	for i := 0; i < n; i++ {
		x = 3.8 * x * (1.0 - x)
	}
	return x
}

func hannWindow(i, length int) float64 {
	return 0.5 * (1.0 - math.Cos(2.0*math.Pi*float64(i)/float64(length-1)))
}

func main() {
	file, err := os.Open("gazo.png")
	if err != nil {
		fmt.Printf("画像の読み込みに失敗しました: %v\n", err)
		return
	}
	defer file.Close()

	img, _, err := image.Decode(file)
	if err != nil {
		fmt.Printf("画像のデコードに失敗しました: %v\n", err)
		return
	}

	bounds := img.Bounds()
	// 元コード通り、画像の縦横幅を取得
	width := bounds.Dx()
	height := bounds.Dy()

	var audioBuffer []int16
	count := 0

	// 元の Racket コード (for* ([y (in-range h)] [x (in-range w)])) の順序通り
	// width と height を使用して0からループを回します
	for y := 0; y < height; y++ {
		for x := 0; x < width; x++ {
			count++

			// 座標のズレを防ぐため bounds.Min を考慮してピクセルを取得
			color := img.At(bounds.Min.X+x, bounds.Min.Y+y)
			r64, g64, b64, _ := color.RGBA()
			r := float64(r64 >> 8)
			g := float64(g64 >> 8)
			b := float64(b64 >> 8)

			rgbSum := r + g + b

			jitter := logisticMap(count, 0.5) * 100.0
			lenJitter := logisticMap(count, 0.5) * 10000.0

			freq := 10.0 + ((rgbSum / 765.0) * 800.0) + jitter

			toneLength := int(11025.0 + lenJitter)
			if toneLength < 1 {
				toneLength = 1
			}

			envLength := toneLength / 32
			if envLength < 1 {
				envLength = 1
			}

			for i := 0; i < toneLength; i++ {
				angle := 2.0 * math.Pi * freq * float64(i) / float64(sampleRate)
				sample := math.Sin(angle) * 0.2

				if i < envLength {
					sample *= hannWindow(i, envLength)
				}

				val := int16(sample * 32767.0)
				audioBuffer = append(audioBuffer, val)
			}
		}
	}

	outFile, err := os.Create("gazo.wav")
	if err != nil {
		fmt.Printf("WAVファイルの作成に失敗しました: %v\n", err)
		return
	}
	defer outFile.Close()

	numChannels := uint16(1)
	bitsPerSample := uint16(16)
	subChunk2Size := uint32(len(audioBuffer) * 2)
	chunkSize := 36 + subChunk2Size
	byteRate := uint32(sampleRate) * uint32(numChannels) * uint32(bitsPerSample/8)
	blockAlign := numChannels * (bitsPerSample / 8)

	outFile.Write([]byte("RIFF"))
	binary.Write(outFile, binary.LittleEndian, chunkSize)
	outFile.Write([]byte("WAVE"))

	outFile.Write([]byte("fmt "))
	binary.Write(outFile, binary.LittleEndian, uint32(16))
	binary.Write(outFile, binary.LittleEndian, uint16(1))
	binary.Write(outFile, binary.LittleEndian, numChannels)
	binary.Write(outFile, binary.LittleEndian, uint32(sampleRate))
	binary.Write(outFile, binary.LittleEndian, byteRate)
	binary.Write(outFile, binary.LittleEndian, blockAlign)
	binary.Write(outFile, binary.LittleEndian, bitsPerSample)

	outFile.Write([]byte("data"))
	binary.Write(outFile, binary.LittleEndian, subChunk2Size)

	err = binary.Write(outFile, binary.LittleEndian, audioBuffer)
	if err != nil {
		fmt.Printf("音声データの書き込みに失敗しました: %v\n", err)
		return
	}

	fmt.Println("gazo.wav の生成が完了しました！")
}

