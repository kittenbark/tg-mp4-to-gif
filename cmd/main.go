package main

import (
	"context"
	"fmt"
	"github.com/kittenbark/tg"
	"os"
	"os/exec"
)

func main() {
	tg.NewFromEnv().
		Scheduler().
		OnError(tg.OnErrorLog).
		Filter(tg.OnPrivate).
		Command("/start", onStart).
		Branch(tg.OnVideo, tg.Synced(onVideo)).
		Start()
}

func onVideo(ctx context.Context, upd *tg.Update) error {
	msg := upd.Message
	asReply := &tg.OptSendMessage{ReplyParameters: tg.AsReplyTo(msg)}
	progress, _ := tg.SendMessage(ctx, msg.Chat.Id, "Downloading...", asReply)
	defer func() {
		if progress != nil {
			_, _ = tg.DeleteMessage(ctx, progress.Chat.Id, progress.MessageId)
		}
	}()

	if msg.Video.FileSize > (20 << 20) {
		_, err := tg.SendMessage(ctx, msg.Chat.Id, "this video is too big to be a gif (sorry)", asReply)
		return err
	}

	vid, err := msg.Video.DownloadTemp(ctx)
	if err != nil {
		return err
	}
	defer os.Remove(vid)

	progressEditOpt := &tg.OptEditMessageText{ChatId: progress.Chat.Id, MessageId: progress.MessageId}
	if progress != nil {
		progress, _ = tg.EditMessageText(ctx, "Converting...", progressEditOpt)
	}
	result, err := makeGif(vid)
	if err != nil {
		return err
	}
	defer os.Remove(result)

	if progress != nil {
		progress, _ = tg.EditMessageText(ctx, "Uploading...", progressEditOpt)
	}
	_, err = tg.SendAnimation(ctx, msg.Chat.Id, tg.FromDisk(result, "@heilmeh.mp4"), &tg.OptSendAnimation{ReplyParameters: tg.AsReplyTo(msg)})
	if err != nil {
		return err
	}
	return nil
}

func onStart(ctx context.Context, upd *tg.Update) error {
	msg := upd.Message
	resp, err := tg.SendMessage(
		ctx,
		msg.Chat.Id,
		"hey, if smth doesnt work, ping @heilmeh",
		&tg.OptSendMessage{ReplyParameters: tg.AsReplyTo(msg)},
	)
	if err != nil {
		return err
	}
	if _, err = tg.PinChatMessage(ctx, resp.Chat.Id, resp.MessageId, &tg.OptPinChatMessage{DisableNotification: true}); err != nil {
		return err
	}
	return nil
}

func makeGif(source string) (result string, err error) {
	result = source + "_converted.mp4"
	cmd := exec.Command("ffmpeg",
		"-i", source,
		"-c:v", "libx264",
		"-an",
		"-vf", "scale='min(1280,iw)':'min(1280,ih)':force_original_aspect_ratio=decrease",
		"-y",
		result,
	)

	if output, err := cmd.CombinedOutput(); err != nil {
		return "", fmt.Errorf("ffmpeg conversion failed: %v\nOutput: %s", err, string(output))
	}

	return result, nil
}
