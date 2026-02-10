package main

import (
	"bufio"
	"fmt"
	"handler/internal/downloader"
	"handler/internal/model"
	"handler/internal/provider"
	"handler/internal/provider/kuwo"
	"handler/internal/server"
	"os"
	"strconv"
	"strings"
)

const (
	pageSize   = 20
	bannerText = `
 __  __           _        _____ ___ _       _               
|  \/  |_   _ ___(_) ___  |  ___|/ _ \ |_ ___| |__   ___ _ __ 
| |\/| | | | / __| |/ __| | |_  |  __/ __/ __| '_ \ / _ \ '__|
| |  | | |_| \__ \ | (__  |  _| | |__| || (__| | | |  __/ |   
|_|  |_|\__,_|___/_|\___| |_|    \___|\__\___|_| |_|\___|_|   
`
)

func main() {
	p := kuwo.New()

	// 如果传了 --web 参数，启动 Web 服务
	for _, arg := range os.Args[1:] {
		if arg == "--web" || arg == "-w" {
			addr := "127.0.0.1:8080"
			// 支持 --web :9090 形式指定端口
			for i, a := range os.Args[1:] {
				if (a == "--web" || a == "-w") && i+2 < len(os.Args) {
					addr = os.Args[i+2]
				}
			}
			fmt.Print(bannerText)
			srv := server.New(p)
			if err := srv.Start(addr); err != nil {
				fmt.Fprintf(os.Stderr, "启动失败: %v\n", err)
				os.Exit(1)
			}
			return
		}
	}

	// 默认 CLI 模式
	runCLI(p)
}

func runCLI(currentProvider provider.Provider) {
	scanner := bufio.NewScanner(os.Stdin)

	fmt.Print(bannerText)
	fmt.Println("  音乐资源获取器 v1.0")
	fmt.Println("  输入关键词搜索歌曲，支持下载")
	fmt.Println(strings.Repeat("─", 50))
	fmt.Printf("\n  当前来源: %s\n", currentProvider.Name())

	for {
		fmt.Print("\n  请输入搜索关键词 (输入 q 退出): ")
		if !scanner.Scan() {
			break
		}
		input := strings.TrimSpace(scanner.Text())

		switch strings.ToLower(input) {
		case "q", "quit", "exit":
			fmt.Println("\n  再见!")
			return
		case "":
			continue
		}

		fmt.Printf("\n  正在搜索: %s ...\n", input)
		songs, err := currentProvider.Search(input, 1, pageSize)
		if err != nil {
			fmt.Printf("  搜索失败: %v\n", err)
			continue
		}
		if len(songs) == 0 {
			fmt.Println("  没有找到相关歌曲")
			continue
		}

		displaySongs(songs)
		handleDownload(songs, currentProvider, scanner)
	}
}

func displaySongs(songs []model.Song) {
	fmt.Println()
	fmt.Printf("  %-4s %-30s %-20s %-20s %-6s\n", "序号", "歌曲名", "歌手", "专辑", "时长")
	fmt.Printf("  %s\n", strings.Repeat("─", 84))

	for i, song := range songs {
		name := truncate(song.Name, 28)
		artist := truncate(song.Artist, 18)
		album := truncate(song.Album, 18)
		fmt.Printf("  %-4d %-30s %-20s %-20s %-6s\n",
			i+1, name, artist, album, song.FormatDuration())
	}

	fmt.Printf("  %s\n", strings.Repeat("─", 84))
	fmt.Printf("  共 %d 首歌曲\n", len(songs))
}

func handleDownload(songs []model.Song, p provider.Provider, scanner *bufio.Scanner) {
	for {
		fmt.Print("\n  请输入要下载的歌曲序号 (多个用逗号分隔, 输入 0 返回搜索): ")
		if !scanner.Scan() {
			return
		}
		input := strings.TrimSpace(scanner.Text())
		if input == "0" || input == "" {
			return
		}

		parts := strings.Split(input, ",")
		for _, part := range parts {
			part = strings.TrimSpace(part)
			idx, err := strconv.Atoi(part)
			if err != nil || idx < 1 || idx > len(songs) {
				fmt.Printf("  无效的序号: %s\n", part)
				continue
			}

			song := songs[idx-1]
			fmt.Printf("\n  正在获取下载链接: %s - %s\n", song.Name, song.Artist)

			playURL, err := p.GetPlayURL(song.ID)
			if err != nil {
				fmt.Printf("  获取下载链接失败: %v\n", err)
				continue
			}

			filename := kuwo.SanitizeFileName(
				fmt.Sprintf("%s - %s.mp3", song.Name, song.Artist))

			savePath, err := downloader.Download(playURL, filename)
			if err != nil {
				fmt.Printf("  下载失败: %v\n", err)
				continue
			}
			fmt.Printf("  下载完成: %s\n", savePath)
		}
		return
	}
}

func truncate(s string, maxLen int) string {
	runes := []rune(s)
	if len(runes) <= maxLen {
		return s
	}
	return string(runes[:maxLen-3]) + "..."
}
