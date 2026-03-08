# iPod AAC Compatibility

The iPod 4th generation (monochrome) uses a PortalPlayer PP5020 SoC with a Wolfson WM8975 audio codec. Its hardware AAC decoder only supports a subset of the AAC standard, which requires specific ffmpeg settings to produce compatible files.

## Required ffmpeg flags

```
-vn
-c:a aac
-profile:a aac_low
-aac_pns 0
-ar 44100
-ac 2
-movflags +faststart
-f ipod
```

### Flag explanations

| Flag | Purpose |
|---|---|
| `-vn` | Strip video/cover art streams. The iPod muxer (`-f ipod`) cannot handle embedded images (e.g. FLAC cover art) and will fail. |
| `-c:a aac` | Use ffmpeg's native AAC encoder. |
| `-profile:a aac_low` | AAC Low Complexity (LC) profile. The iPod only supports AAC-LC — not HE-AAC, HE-AACv2, or AAC-LD. |
| `-aac_pns 0` | Disable Perceptual Noise Substitution. ffmpeg enables PNS by default, but the iPod's hardware decoder does not support it. PNS causes audible chirps/artifacts during playback. |
| `-ar 44100` | 44.1 kHz sample rate. Standard for music; the iPod handles this natively. |
| `-ac 2` | Stereo output. |
| `-movflags +faststart` | Move the MP4 moov atom to the beginning of the file for faster playback start on the iPod's slow storage. |
| `-f ipod` | Use the iPod-specific MP4 muxer. This produces a properly signaled M4A container. Raw AAC bitstreams (no container) will not play — the iPod needs the MP4/M4A container with correct atom structure. |

## Problems we hit

### Tracks won't play (skip immediately)

Subsonic's `/rest/stream?format=aac` endpoint returns a raw AAC bitstream, not an M4A file. The iPod can load metadata but immediately skips to the next track because it can't parse a containerless AAC stream.

**Fix**: Download the original file via `/rest/download` and transcode locally with `-f ipod` to produce a proper M4A container.

### Audio chirps/artifacts

ffmpeg's native AAC encoder enables Perceptual Noise Substitution (PNS) by default. PNS replaces low-energy spectral bands with parametric noise, which is valid per the AAC spec but unsupported by the iPod's Wolfson WM8975 decoder. The result is short, sharp chirps scattered throughout playback.

**Fix**: `-aac_pns 0` disables PNS entirely.

### ffmpeg fails on FLAC with cover art

FLAC files often contain embedded cover art (MJPEG). When ffmpeg tries to mux this into the iPod format, it fails because the iPod muxer doesn't support video streams.

**Fix**: `-vn` strips all video/image streams before muxing.

## Bitrate

256 kbps AAC-LC is the default. The iPod supports up to 320 kbps AAC but 256k is a good balance of quality and file size for the limited storage.
