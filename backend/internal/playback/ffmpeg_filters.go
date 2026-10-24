package playback

import (
	"fmt"
	"strconv"
	"strings"
	"time"

	"retrotv/internal/presentation"
)

type FFmpegPresentationFilter struct {
	InputPaths  []string
	FilterGraph string
	OutputLabel string
}

func BuildFFmpegPresentationFilter(config presentation.BroadcastPresentation) FFmpegPresentationFilter {
	builder := ffmpegFilterBuilder{
		currentLabel: "0:v",
		outputLabel:  "vout",
	}
	if config.ChannelBug != nil {
		builder.addImageOverlay(*config.ChannelBug, "bug")
	}
	if config.Rating != nil {
		builder.addImageOverlay(*config.Rating, "rating")
	}
	if config.NowPlaying != nil {
		builder.addTextOverlay(*config.NowPlaying)
	}
	if len(builder.steps) == 0 {
		return FFmpegPresentationFilter{OutputLabel: "0:v"}
	}
	builder.steps = append(builder.steps, fmt.Sprintf("[%s]null[%s]", builder.currentLabel, builder.outputLabel))
	return FFmpegPresentationFilter{
		InputPaths:  builder.inputPaths,
		FilterGraph: strings.Join(builder.steps, ";"),
		OutputLabel: builder.outputLabel,
	}
}

type ffmpegFilterBuilder struct {
	inputPaths   []string
	steps        []string
	currentLabel string
	outputLabel  string
	nextLabel    int
}

func (b *ffmpegFilterBuilder) addImageOverlay(overlay presentation.ImageOverlay, name string) {
	b.inputPaths = append(b.inputPaths, overlay.AssetPath)
	inputIndex := len(b.inputPaths)
	imageLabel := fmt.Sprintf("%s%d", name, inputIndex)
	scaledLabel := fmt.Sprintf("%s_scaled%d", name, inputIndex)
	next := b.nextVideoLabel()

	width := overlay.Width
	if width <= 0 {
		width = 160
	}
	opacity := overlay.Opacity
	if opacity <= 0 || opacity > 1 {
		opacity = 1
	}
	x, y := overlayCoordinates(overlay.Position, overlay.MarginX, overlay.MarginY, "overlay_w", "overlay_h")
	enable := durationEnable(overlay.Duration)

	b.steps = append(b.steps,
		fmt.Sprintf("[%d:v]format=rgba[%s]", inputIndex, imageLabel),
		fmt.Sprintf("[%s]scale=%d:-1,colorchannelmixer=aa=%s[%s]", imageLabel, width, formatFloat(opacity), scaledLabel),
		fmt.Sprintf("[%s][%s]overlay=x=%s:y=%s%s[%s]", b.currentLabel, scaledLabel, x, y, enable, next),
	)
	b.currentLabel = next
}

func (b *ffmpegFilterBuilder) addTextOverlay(overlay presentation.TextOverlay) {
	next := b.nextVideoLabel()
	marginX := overlay.MarginX
	if marginX <= 0 {
		marginX = 48
	}
	marginY := overlay.MarginY
	if marginY <= 0 {
		marginY = 48
	}
	x, y := overlayCoordinates(overlay.Position, marginX, marginY, "text_w", "text_h")
	enable := durationEnable(overlay.Duration)
	b.steps = append(b.steps, fmt.Sprintf(
		"[%s]drawtext=text='%s':x=%s:y=%s:fontsize=36:fontcolor=white:box=1:boxcolor=black@0.65:boxborderw=16%s[%s]",
		b.currentLabel,
		EscapeDrawtext(overlay.Text),
		x,
		y,
		enable,
		next,
	))
	b.currentLabel = next
}

func (b *ffmpegFilterBuilder) nextVideoLabel() string {
	b.nextLabel++
	return fmt.Sprintf("v%d", b.nextLabel)
}

func overlayCoordinates(position presentation.OverlayPosition, marginX int, marginY int, widthName string, heightName string) (string, string) {
	if marginX < 0 {
		marginX = 0
	}
	if marginY < 0 {
		marginY = 0
	}
	xMargin := strconv.Itoa(marginX)
	yMargin := strconv.Itoa(marginY)
	switch position {
	case presentation.PositionTopLeft:
		return xMargin, yMargin
	case presentation.PositionTopRight:
		return "main_w-" + widthName + "-" + xMargin, yMargin
	case presentation.PositionBottomLeft:
		return xMargin, "main_h-" + heightName + "-" + yMargin
	default:
		return "main_w-" + widthName + "-" + xMargin, "main_h-" + heightName + "-" + yMargin
	}
}

func durationEnable(duration time.Duration) string {
	if duration <= 0 {
		return ""
	}
	return ":enable='between(t,0," + formatFloat(duration.Seconds()) + ")'"
}

func EscapeDrawtext(value string) string {
	replacer := strings.NewReplacer(
		`\`, `\\`,
		`:`, `\:`,
		`'`, `\'`,
		`%`, `\%`,
		`[`, `\[`,
		`]`, `\]`,
		`,`, `\,`,
	)
	return replacer.Replace(value)
}

func formatFloat(value float64) string {
	return strconv.FormatFloat(value, 'f', -1, 64)
}
