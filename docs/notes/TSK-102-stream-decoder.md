# Task Specification & Architecture Note: TSK-102 (Opus/AAC Stream Demuxer & Decoder)

**Workstream:** `WS-AUD`  
**File Scope:** `internal/audio/decoder.go`, `internal/audio/decoder_test.go`  
**Dependencies:** `TSK-101`  
**Status:** `DONE`  
**Package:** `chordly/internal/audio`  

---

## 1. Executive Summary & Objective

The **Opus/AAC Stream Demuxer & Decoder (`TSK-102`)** implements Chordly's pure-Go container demuxing and decoding pipeline. It bridges streamed network chunks or cached disk bytes (`TSK-108`, `TSK-103`) directly into continuous 48kHz, 16-bit signed stereo PCM frames consumed by the audio output driver (`TSK-101`).

---

## 2. Supported Container & Codec Formats

| Container / Stream | Codec | InnerTube Format Target | Magic Sniff Header |
| :--- | :--- | :--- | :--- |
| **WebM / Matroska** | Opus (`A_OPUS`) | itag 251 (~160kbps) | `0x1A 0x45 0xDF 0xA3` (EBML Header) |
| **ISO BMFF / MP4** | AAC-LC | itag 140 (~128kbps) | `....ftyp` (M4A box) |
| **ADTS AAC** | AAC | Live AAC radio streams | `0xFF 0xF1` / `0xFF 0xF9` |
| **Ogg** | Opus | Fallback Ogg streams | `"OggS"` |
| **PCM / WAV** | Raw Linear PCM | Local test fixtures | `"RIFF....WAVE"` |

---

## 3. Architecture & Data Flow

```
[HTTP Stream / Disk Cache]
             │
             ▼
     DetectFormat() (Transparent 32-byte header inspection)
             │
             ▼
 ┌───────────────────────┐
 │ StreamDecoder         │
 │  ├── webmDemuxer      │ ──► Parses EBML VINTs, Cluster timecodes & SimpleBlocks
 │  ├── mp4Demuxer       │ ──► Parses ISO boxes (moov, trak, mdat)
 │  └── adtsDemuxer      │ ──► Parses ADTS syncword & frame headers
 └───────────┬───────────┘
             │
             ▼
    decodePacketToPCM() (Converts packet to 48kHz 16-bit stereo PCM)
             │
             ▼
    io.Reader interface (Read pcm frames)
             │
             ▼
     Device.Play() (Audio output hardware)
```

---

## 4. Concurrency & Thread Safety

- **Serialized Read Protection:** `StreamDecoder` protects internal PCM buffers and demuxer state with `sync.Mutex`.
- **Race-Free Device Integration:** `Device` coordinates `src.Read()` calls under mutex lock (`srcMu`) ensuring safe transitions during high-frequency Play/Pause/Seek operations.

---

## 5. Verification Matrix

- `TestDetectFormat`: Verified magic-byte sniffing across WebM, Ogg, MP4, ADTS, WAV, and fallback.
- `TestWebMDemuxer`: Verified EBML element parsing, VINT calculation, and SimpleBlock extraction.
- `TestADTSDemuxer`: Verified ADTS syncword, profile, and frame length demuxing.
- `TestDecoderFeedToDevice`: End-to-end integration verifying that `Decoder` feeds valid 48kHz PCM frames directly into `Device`.
- `TestDecoderClose`: Verified resource cleanup and `io.ErrClosedPipe` return on closed stream reads.
- **Race Detector:** `go test -v -race ./internal/audio/...` passed with 0 data races.
- **Full Project Build:** `go build ./...` compiled with 0 errors.
