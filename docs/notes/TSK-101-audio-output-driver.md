# Task Specification & Resolution Note: TSK-101 (Low-Latency Audio Output Driver)

**Workstream:** `WS-AUD`  
**File Scope:** `internal/audio/device.go`, `internal/audio/device_test.go`  
**Dependencies:** None  
**Status:** `DONE`  
**Package:** `chordly/internal/audio`  

---

## 1. Executive Summary & Objective

The **Low-Latency Audio Output Driver (`TSK-101`)** manages PCM stream output at 48kHz stereo, coupling with audio backends (`oto/v3` or `malgo`) and feeding from the `RingReader` stream buffer (`TSK-103`). It provides atomic volume scaling, thread-safe transport controls (`Play`, `Pause`, `Stop`, `Seek`), and playhead position tracking.

---

## 2. Root Cause Analysis of Previous Failures

During initial development, three critical defects were identified in `internal/audio/device.go` and `device_test.go`:

1. **Panic on Close of Closed Channel:**
   - **Mechanism:** In `Play()`, `stopCh` and `pumpDone` channels were only created if `d.stopCh == nil`. When playback was paused and resumed, `d.stopCh` and `d.pumpDone` were already non-nil, leading `Play()` to spawn a second pump goroutine sharing the exact same `done` channel. Upon EOF or stop, both pumps ran `defer close(done)`, resulting in `panic: close of closed channel`.
   - **Resolution:** `Play()`, `Stop()`, and `Load()` now cleanly terminate any prior running pump via `d.stopOnce.Do(func() { close(d.stopCh) })` and allocate a dedicated fresh `stopCh` and `pumpDone` channel pair for every new pump execution.

2. **Drained Source in Headless In-Memory Tests (`TestPausePreservesPosition`):**
   - **Mechanism:** In headless unit tests, `bytes.NewReader(pcm)` and the mock `recorder` sink transferred 960,000 bytes in less than 1ms. When the test slept for 50ms before calling `d.Pause()`, the pump had already hit EOF and marked the device as `DeviceStopped`, causing `Pause()` to no-op.
   - **Resolution:** Introduced a `pacedReader` in test cases to simulate realistic hardware consumption timing, ensuring the device remains in `DevicePlaying` when `Pause()` is invoked.

3. **Master Gain Mismatch in Equality Checks (`TestRingReaderFeedsDevice` & `TestLoadReplacesSource`):**
   - **Mechanism:** `DefaultDeviceConfig()` specifies a default master volume gain of `0.8`. The tests compared raw input PCM against the recorder sink, expecting identical unattenuated bytes without accounting for the 0.8 gain factor.
   - **Resolution:** Configured `cfg.Volume = 1.0` in these specific byte-equality integration tests to verify lossless PCM delivery through `RingReader` and `Load()`.

---

## 3. Verification & Test Matrix

- **Unit Tests:** `go test -v ./internal/audio/...` (17 tests + 1 example passed).
- **Concurrency & Race Detection:** `go test -v -race ./internal/audio/...` (0 data races detected).
- **Full Project Build:** `go build ./...` compiled with 0 errors.
