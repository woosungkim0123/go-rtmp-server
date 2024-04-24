package internal

import (
	"bufio"
	"fmt"
	"io"
	"os"
	"os/exec"
)

var HLSOutputBasePath = "/hls-preview/"

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
	cmd := exec.Command("ffmpeg",
		"-v", "debug", // 'verbose' 대신 원래 명령의 'debug' 레벨을 사용
		"-i", "rtmp://localhost/"+c.AppName+"/"+streamKey,
		"-c:v", "libx264",
		"-c:a", "aac",
		"-ac", "1",
		"-strict", "-2",
		"-crf", "18",
		"-profile:v", "baseline",
		"-maxrate", "400k",
		"-bufsize", "1835k",
		"-pix_fmt", "yuv420p",
		"-flags", "-global_header",
		"-hls_time", "3",
		"-hls_list_size", "6",
		"-hls_flags", "delete_segments", // HLS 플래그에 'delete_segments' 추가
		"-start_number", "1",
		output+"/index.m3u8",
	)
	//cmd := exec.Command("ffmpeg", "-version")

	//fmt.Println("Generating preview for", streamKey, c.AppName)
	//cmd := exec.Command("ffmpeg", "-v", "verbose", "-i", "rtmp://localhost/"+c.AppName+"/"+streamKey, "-c:v", "libx264", "-c:a", "aac", "-ac", "1", "-strict", "-2", "-crf", "18", "-profile:v", "baseline", "-maxrate", "400k", "-bufsize", "1835k", "-pix_fmt", "yuv420p", "-flags", "-global_header", "-hls_time", "3", "-hls_list_size", "6", "-hls_wrap", "10", "-start_number", "1", output+"/index.m3u8")
	stdoutPipe, err := cmd.StdoutPipe()
	if err != nil {
		panic(err)
	}
	stderrPipe, err := cmd.StderrPipe()
	if err != nil {
		panic(err)
	}

	if err := cmd.Start(); err != nil {
		panic(err)
	}

	// 표준 출력 로그 처리
	go readOutput(stdoutPipe)
	// 표준 에러 로그 처리
	go readOutput(stderrPipe)

	// 커맨드 완료 대기
	if err := cmd.Wait(); err != nil {
		fmt.Println("FFmpeg failed:", err)
	}

	fmt.Println("FFMPEG DONE")
}

func readOutput(pipe io.ReadCloser) {
	scanner := bufio.NewScanner(pipe)
	for scanner.Scan() {
		fmt.Println(scanner.Text())
	}
	if err := scanner.Err(); err != nil {
		fmt.Println("Error reading from pipe:", err)
	}
}
