package models

import "testing"

func TestParseTranscodeProgress(t *testing.T) {
	sampleLines := []string{
		`frame= 1147 fps= 71 q=-1.0 Lsize=    8936kB time=00:00:45.88 bitrate=1595.3kbits/s speed=2.82x    
video:8179kB audio:721kB subtitle:0kB other streams:0kB global headers:0kB muxing overhead: 0.398820% `,
		`frame=   53 fps=0.0 q=0.0 size=       0kB time=00:00:02.36 bitrate=   0.2kbits/s speed=4.69x    
`,
		`frame= 3495 fps=109 q=-1.0 Lsize=   26121kB time=00:02:19.81 bitrate=1530.4kbits/s speed=4.37x    
video:23824kB audio:2193kB subtitle:0kB other streams:0kB global headers:0kB muxing overhead: 0.402554%
`,
	}

}
