package internal

import (
	"fmt"
	"log"
	"os"
	"os/exec"
)

var (
	HLSOutputBasePath = "/hls-preview/"
)

func InitPreviewServer(ctx *StreamContext) {
	for {
		select {
		case streamKey := <-ctx.Preview:
			go makeHls(ctx, streamKey)
		}
	}
}

func makeHls(ctx *StreamContext, streamKey string) {
	c, ok := ctx.Sessions[streamKey]
	if !ok {
		fmt.Println("Not Found", streamKey)
	}
	output := HLSOutputBasePath + streamKey
	if _, err := os.Stat(output); err != nil {
		if os.IsNotExist(err) {
			os.MkdirAll(output, os.ModePerm)
		}
	}

	f, err := os.Create(output + "/index.m3u8")
	if err != nil {
		panic(err)
	}
	f.Write([]byte("#EXTM3U\n#EXT-X-VERSION:3\n#EXT-X-TARGETDURATION:8\n#EXT-X-MEDIA-SEQUENCE:1"))
	f.Close()

	fmt.Println("Generating preview for", streamKey, c.AppName)
	cmd := exec.Command("ffmpeg", "-v", "verbose", "-i", "rtmp://localhost/"+c.AppName+"/"+streamKey, "-c:v", "libx264", "-c:a", "aac", "-ac", "1", "-strict", "-2", "-crf", "18", "-profile:v", "baseline", "-maxrate", "400k", "-bufsize", "1835k", "-pix_fmt", "yuv420p", "-flags", "-global_header", "-hls_time", "3", "-hls_list_size", "6", "-hls_wrap", "10", "-start_number", "1", output+"/index.m3u8")
	err = cmd.Run()
	os.RemoveAll(output)
	if err != nil {
		log.Fatal(err)
	}
	fmt.Println("FFMPEG DONE")
}
