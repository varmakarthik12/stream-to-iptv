# stream-to-iptv
Stream to IPTV coverts various media input sources into IPTV-compatible HLS format. The tool supports a wide range of input sources including local files, HTTP/HTTPS URLs, RTMP streams, UDP streams, RTSP streams, and device inputs.

**Features**:
- Convert any media source to IPTV-compatible HLS format.
- Supports multiple input sources: local files, HTTP/HTTPS URLs, RTMP, UDP, RTSP, and device inputs.
- Configurable HLS segment duration and playlist size.
- Automatically manages storage by deleting old segments.
- Creates M3U IPTV playlists.
- Easy to integrate and extend.

**Requirements**:
- [FFMpeg](https://www.ffmpeg.org/download.html)

**Usage**:
1. Clone the repository:
   ```shell
   git clone https://github.com/yourusername/stream-to-iptv.git
   ```
2. Navigate to the project directory:
   ```shell
   cd stream-to-iptv
   ```
3. Run stream to IPTV.
  ```shell
  go run ./cmd/main.go --config <path to config.json>
  ```



**Installation**: (TODO - Support Github releases)
1. Clone the repository:
   ```shell
   git clone https://github.com/yourusername/stream-to-iptv.git
   ```
2. Navigate to the project directory:
   ```shell
   cd stream-to-iptv
   ```
3. Build the project:
   ```shell
   ./build.sh
   ```
4. Builds are generated in the `build` folder. Use a build that works for your OS / Platform
  ```shell
    ./stream -config <path to config.json>
   ```

**Contributing**:
Contributions are welcome! Please open an issue or submit a pull request for any improvements or bug fixes.
