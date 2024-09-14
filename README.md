# stream-to-iptv
Stream to IPTV converts various media input sources into IPTV-compatible HLS format. The tool supports a wide range of input sources including local files, HTTP/HTTPS URLs, RTMP streams, UDP streams, RTSP streams, and device inputs.

## Features:
- Convert any media source to IPTV-compatible HLS format.
- Supports multiple input sources: local files, HTTP/HTTPS URLs, RTMP, UDP, RTSP, and device inputs.
- Configurable HLS segment duration and playlist size.
- Automatically manages storage by deleting old segments.
- Creates M3U IPTV playlists.
- Easy to integrate and extend.

**Requirements**:
- The host machine should have [FFmpeg](https://www.ffmpeg.org/download.html) installed
 
## Dev Setup
  `yarn dev` and you should be good to go.

## Installation: 
(TODO - Support Github releases)
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
   yarn build
   ```
4. Builds are generated in the `build` folder. Use a build that works for your OS / Platform
   ```shell
      ./stream -config <path to config.json>
   ```
### CLI Inputs

| Variable     | Description                                      | Default Value |
|--------------|--------------------------------------------------|---------------|
| `CONFIG_FILE`| The path to the configuration file.|./config.json|


### Environment Variables

The following environment variables can be used to configure the application:

| Variable     | Description                                      | Default Value |
|--------------|--------------------------------------------------|---------------|
| `PORT`       | The port on which the server will listen.        | `8068`        |
| `CONFIG_FILE`| The path to the configuration file. Same as `g` flag. but as a env var             |          |
| `MAX_SEGMENTS_COUNT`  | The maximum number of live segments to keep.     | `10`        |
| `MAX_SEGMENT_TIME`  | The maximum time in each segment.         | `10`        |
| `EPG_URL`           | The URL for the Electronic Program Guide (EPG).  |https://avkb.short.gy/epg.xml.gz |
| `IP_ADDR`           | Ip address of the network interface broadcasting the network stream. Same as `ffmpeg -i XXXXX?localaddr=<IP_ADDR>`  | |
| `BUFFER_SIZE` | Customizable buffer size for unstable network streams. Increase this on jitters | `1000000`|
| `USE_GPU` | Use Nvidia CUDA GPU for hardware acceleration | `false`|


### `config.json` Documentation

| Field  | Type   | Description                                                                 | Example Values                                                                 |
|--------|--------|-----------------------------------------------------------------------------|--------------------------------------------------------------------------------|
| name   | string | The name of the channel.                                                    | `"My Channel"`                                                                 |
| media  | string | The media source for the stream. Possible values include:                   | `"udp://239.0.0.1:1234"`, `"http://example.com/stream.m3u8"`, `"file:///path/to/video.mp4"` |
|        |        | - **UDP**: A UDP stream URL.                                                | ` "udp://@239.255.255.250:1234"`                                                       |
|        |        | - **HLS**: An HTTP Live Streaming (HLS) URL.                                | `"http://example.com/stream.m3u8"`                                             |
|        |        | - **HTTP**: A direct HTTP URL to a media file.                              | `"http://example.com/video.mp4"`                                               |
|        |        | - **File**: A local file path.                                              | `"file:///path/to/video.mp4"`                                                  |
| logo   | string | The URL or path to the channel's logo image. This will be reference in IPTV playlist.                                | `"http://example.com/logo.png"`, `"file:///path/to/logo.png"`                  |
| program_id | string | A unique identifier for the program. | `"12345"` |
| groups | string array | An array of group names to which the channel belongs. | `["Kids", "Entertainment"]` |
| tvg_id    | string       | The TV Guide ID for the channel. (Dependent on `EPG_URL` ENV var) | `"my_channel_tvg_id"`|



**Example** `config.json`

```json
[
   {
   "Name": "My Channel",
   "Media": "http://example.com/stream.m3u8",
   "Logo": "http://example.com/logo.png"
   }
]
```


### Run Docker Container

To run the Docker container, use the following command:

```sh
docker run -p 8068:8068 -v path/to/config/config.json:/app/config.json varmakarthik12/stream-to-iptv
```

### Docker Compose Usage

You can also use Docker Compose to build and run the project. 

1. Ensure you have `docker-compose.yml` .
```
services:
  stream-to-iptv:
    image: varmakarthik12/stream-to-iptv
    ports:
      - "8068:8068"
    volumes:
      - path/to/config/config.json:/app/config.json
```
2. Run the following command to start the service:

```sh
docker-compose up -d
```

### API Endpoints

#### `GET /playlist.m3u`

This API responds with an IPTV playlist. It constructs this using the `Config.json` struct, utilizing the `groups` , `logo` & other fields in the to create the IPTV playlist.

Example Request:
```
Get http://<host>:<port>/playlist.m3u
```

Example response:
```m3u
#EXTM3U
#EXTINF:-1 tvg-logo="http://example.com/logo.png" group-title="Some Group",Some TV
http://localhost:8068/stream/Some%20TV/Some%20TV.m3u8
#EXTINF:-1 tvg-logo="http://example.com/logo.png" group-title="Another Group",TestTv
http://localhost:8068/stream/TestTv/TestTv.m3u8
```


## Simulate a RDP Stream
```shell
ffmpeg -re -f lavfi -i testsrc=size=640x360:rate=30 -f lavfi -i sine=frequency=1000 -c:v libx264 -preset ultrafast -c:a aac -f mpegts -max_delay 500000 -bufsize 1000k udp://239.255.255.250:1234
```

**Contributing**:
Contributions are welcome! Please open an issue or submit a pull request for any improvements or bug fixes.
