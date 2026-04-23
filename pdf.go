package main

import (
	"fmt"
	"image"
	"image/color"
	"image/draw"
	"image/jpeg"
	"os"
	"path/filepath"
	"time"

	"github.com/klippa-app/go-pdfium"
	"github.com/klippa-app/go-pdfium/requests"
	"github.com/klippa-app/go-pdfium/webassembly"
)

// pdfiumPool holds the WebAssembly instance globally to avoid multiple initializations.
var pdfiumPool pdfium.Pool

/*
getPdfiumPool returns the active PDFium pool or initializes it
(lazy initialization) if it does not exist yet.
*/
func getPdfiumPool() (pdfium.Pool, error) {
	if pdfiumPool != nil {
		return pdfiumPool, nil
	}

	pool, err := webassembly.Init(webassembly.Config{
		MinIdle:  1,
		MaxIdle:  1,
		MaxTotal: 1,
	})
	if err != nil {
		return nil, fmt.Errorf("error initializing wazero/pdfium pool: %w", err)
	}

	pdfiumPool = pool
	return pdfiumPool, nil
}

/*
closePdfiumPool closes the global PDFium pool at the end of the program.
*/
func closePdfiumPool() {
	if pdfiumPool != nil {
		_ = pdfiumPool.Close()
		pdfiumPool = nil // prevents errors from accidental double calls via defer
	}

	// completely delete temporary directory for generated PDF images (data privacy)
	pdfTempDir := filepath.Join(".", ".tmp-pdf-images")
	_ = os.RemoveAll(pdfTempDir)
}

/*
convertPDFToImages reads a PDF file, converts each page into a JPG, and saves them
in the specified output directory. Returns the number of converted pages.
*/
func convertPDFToImages(pdfPath string, outDir string, dpi int, quality int) (int, error) {
	pool, err := getPdfiumPool()
	if err != nil {
		return 0, err
	}

	if err := os.MkdirAll(outDir, 0750); err != nil {
		return 0, fmt.Errorf("error creating output directory: %w", err)
	}

	instance, err := pool.GetInstance(time.Second * 15)
	if err != nil {
		return 0, fmt.Errorf("error creating PDFium instance: %w", err)
	}
	defer func() { _ = instance.Close() }()

	pdfData, err := os.ReadFile(pdfPath)
	if err != nil {
		return 0, fmt.Errorf("could not read PDF: %w", err)
	}

	doc, err := instance.OpenDocument(&requests.OpenDocument{
		File: &pdfData,
	})
	if err != nil {
		return 0, fmt.Errorf("error opening PDF in PDFium: %w", err)
	}
	defer func() {
		_, _ = instance.FPDF_CloseDocument(&requests.FPDF_CloseDocument{
			Document: doc.Document,
		})
	}()

	pageCountRes, err := instance.FPDF_GetPageCount(&requests.FPDF_GetPageCount{
		Document: doc.Document,
	})
	if err != nil {
		return 0, fmt.Errorf("error getting page count: %w", err)
	}

	totalPages := pageCountRes.PageCount
	baseFileName := filepath.Base(pdfPath)
	convertedCount := 0

	for i := 0; i < totalPages; i++ {
		renderRes, err := instance.RenderPageInDPI(&requests.RenderPageInDPI{
			DPI: dpi,
			Page: requests.Page{
				ByIndex: &requests.PageByIndex{
					Document: doc.Document,
					Index:    i,
				},
			},
		})
		if err != nil {
			return convertedCount, fmt.Errorf("error rendering page %d: %w", i+1, err)
		}

		renderedImg := renderRes.Result.Image
		bounds := renderedImg.Bounds()
		whiteCanvas := image.NewRGBA(bounds)
		draw.Draw(whiteCanvas, bounds, &image.Uniform{C: color.White}, image.Point{}, draw.Src)
		draw.Draw(whiteCanvas, bounds, renderedImg, image.Point{}, draw.Over)

		// format: originalname.pdf.001.jpg for chronologically correct sorting
		fileName := fmt.Sprintf("%s.%03d.jpg", baseFileName, i+1)
		outFilePath := filepath.Join(outDir, fileName)

		outFile, err := os.Create(outFilePath)
		if err != nil {
			return convertedCount, fmt.Errorf("error creating output file '%s': %w", outFilePath, err)
		}

		options := &jpeg.Options{Quality: quality}
		err = jpeg.Encode(outFile, whiteCanvas, options)
		_ = outFile.Close()

		if err != nil {
			return convertedCount, fmt.Errorf("error encoding JPEG for page %d: %w", i+1, err)
		}
		convertedCount++
	}

	return convertedCount, nil
}
