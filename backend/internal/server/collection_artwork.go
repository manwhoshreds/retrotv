package server

import (
	"context"
	"fmt"
	"image"
	"image/color"
	"image/draw"
	_ "image/gif"
	_ "image/jpeg"
	"image/png"
	"os"
	"path/filepath"
	"strings"
	"time"

	xdraw "golang.org/x/image/draw"
	"golang.org/x/image/font"
	"golang.org/x/image/font/gofont/gobold"
	"golang.org/x/image/font/opentype"
	"golang.org/x/image/math/fixed"

	"retrotv/internal/models"
)

const (
	collectionArtworkWidth  = 1200
	collectionArtworkHeight = 1200
	collectionPosterHeight  = 900
)

func (s *Server) generateCollectionArtwork(ctx context.Context, collection models.Collection) (string, error) {
	if len(collection.ArtworkMedia) != 2 {
		return "", fmt.Errorf("auto artwork requires exactly two media items")
	}
	posters := make([]image.Image, 0, 2)
	for _, media := range collection.ArtworkMedia {
		path, err := s.collectionPosterPath(ctx, media)
		if err != nil {
			return "", err
		}
		file, err := os.Open(absoluteMediaPath(path))
		if err != nil {
			return "", fmt.Errorf("open artwork poster: %w", err)
		}
		poster, _, decodeErr := image.Decode(file)
		closeErr := file.Close()
		if decodeErr != nil {
			return "", fmt.Errorf("decode artwork poster: %w", decodeErr)
		}
		if closeErr != nil {
			return "", closeErr
		}
		posters = append(posters, poster)
	}

	canvas := image.NewRGBA(image.Rect(0, 0, collectionArtworkWidth, collectionArtworkHeight))
	draw.Draw(canvas, canvas.Bounds(), image.NewUniform(color.RGBA{R: 3, G: 8, B: 17, A: 255}), image.Point{}, draw.Src)
	for index, poster := range posters {
		destination := image.Rect(index*collectionArtworkWidth/2, 0, (index+1)*collectionArtworkWidth/2, collectionPosterHeight)
		drawPosterCover(canvas, destination, poster)
	}
	drawCollectionFooter(canvas)
	if err := drawCollectionTitle(canvas, collection.Name); err != nil {
		return "", err
	}

	directory := filepath.Join(s.artworkDir, "collections", "generated")
	if err := os.MkdirAll(directory, 0o755); err != nil {
		return "", err
	}
	filename := fmt.Sprintf("%d.png", collection.ID)
	temporary, err := os.CreateTemp(directory, filename+"-*.tmp")
	if err != nil {
		return "", err
	}
	temporaryPath := temporary.Name()
	defer os.Remove(temporaryPath)
	if err := png.Encode(temporary, canvas); err != nil {
		_ = temporary.Close()
		return "", err
	}
	if err := temporary.Close(); err != nil {
		return "", err
	}
	if err := os.Rename(temporaryPath, filepath.Join(directory, filename)); err != nil {
		return "", err
	}
	return fmt.Sprintf("/artwork/collections/generated/%s?v=%d", filename, time.Now().UnixNano()), nil
}

func (s *Server) collectionPosterPath(ctx context.Context, media models.CollectionMedia) (string, error) {
	switch media.MediaType {
	case "movie":
		movie, err := s.db.MovieByID(ctx, media.MediaID)
		if err != nil {
			return "", err
		}
		return movie.PosterLocalPath, nil
	case "tv":
		show, err := s.db.TVShowByID(ctx, media.MediaID)
		if err != nil {
			return "", err
		}
		return show.PosterLocalPath, nil
	default:
		return "", fmt.Errorf("unsupported artwork media type %q", media.MediaType)
	}
}

func drawPosterCover(destination draw.Image, bounds image.Rectangle, source image.Image) {
	sourceBounds := source.Bounds()
	sourceWidth, sourceHeight := sourceBounds.Dx(), sourceBounds.Dy()
	targetWidth, targetHeight := bounds.Dx(), bounds.Dy()
	if sourceWidth*targetHeight > sourceHeight*targetWidth {
		cropWidth := sourceHeight * targetWidth / targetHeight
		left := sourceBounds.Min.X + (sourceWidth-cropWidth)/2
		sourceBounds = image.Rect(left, sourceBounds.Min.Y, left+cropWidth, sourceBounds.Max.Y)
	} else {
		cropHeight := sourceWidth * targetHeight / targetWidth
		top := sourceBounds.Min.Y
		sourceBounds = image.Rect(sourceBounds.Min.X, top, sourceBounds.Max.X, top+cropHeight)
	}
	xdraw.CatmullRom.Scale(destination, bounds, source, sourceBounds, draw.Over, nil)
}

func drawCollectionFooter(canvas *image.RGBA) {
	const footerHeight = 400
	start := collectionArtworkHeight - footerHeight
	for y := start; y < collectionArtworkHeight; y++ {
		progress := float64(y-start) / float64(footerHeight-1)
		alpha := uint8(progress * 242)
		draw.Draw(canvas, image.Rect(0, y, collectionArtworkWidth, y+1), image.NewUniform(color.RGBA{A: alpha}), image.Point{}, draw.Over)
	}
}

func drawCollectionTitle(canvas *image.RGBA, title string) error {
	parsed, err := opentype.Parse(gobold.TTF)
	if err != nil {
		return err
	}
	title = strings.ToUpper(strings.TrimSpace(title))
	if title == "" {
		title = "UNTITLED COLLECTION"
	}
	var face font.Face
	for size := 86.0; size >= 38; size -= 4 {
		candidate, faceErr := opentype.NewFace(parsed, &opentype.FaceOptions{Size: size, DPI: 72, Hinting: font.HintingFull})
		if faceErr != nil {
			return faceErr
		}
		if face != nil {
			_ = face.Close()
		}
		face = candidate
		if font.MeasureString(face, title).Ceil() <= collectionArtworkWidth-120 {
			break
		}
	}
	defer face.Close()
	width := font.MeasureString(face, title).Ceil()
	metrics := face.Metrics()
	baseline := collectionArtworkHeight - 72 - metrics.Descent.Ceil()
	drawer := font.Drawer{Dst: canvas, Src: image.NewUniform(color.White), Face: face, Dot: fixedPoint((collectionArtworkWidth-width)/2, baseline)}
	drawer.DrawString(title)
	return nil
}

func fixedPoint(x int, y int) fixed.Point26_6 {
	return fixed.P(x, y)
}
