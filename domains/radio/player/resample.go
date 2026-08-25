package player

import (
	"encoding/binary"
	"io"
)

// OutputRate is the single oto context rate (AI-5: one context per process,
// so every stream is resampled to it). NWR relays are 22.05/44.1 kHz.
const OutputRate = 44100

// resampler converts 16-bit little-endian stereo PCM from src Hz to
// OutputRate by linear interpolation — cheap and more than adequate for a
// 32 kbps voice broadcast. Passes through when the rates match.
type resampler struct {
	r       io.Reader
	src     int
	pos     float64 // fractional read position in source frames
	step    float64
	pending []byte // undecoded source bytes carried between reads
	scratch []byte
}

func newResampler(r io.Reader, srcRate int) io.Reader {
	if srcRate == OutputRate || srcRate <= 0 {
		return r
	}
	return &resampler{r: r, src: srcRate, step: float64(srcRate) / OutputRate, scratch: make([]byte, 8192)}
}

func (rs *resampler) Read(p []byte) (int, error) {
	out := 0
	// Bounded per P10-02: one output frame per iteration, at most len(p)/4.
	for i := 0; i < len(p)/4 && out+4 <= len(p); i++ {
		// Need source frames floor(pos) and floor(pos)+1.
		need := int(rs.pos) + 2
		for r := 0; r < 64 && len(rs.pending)/4 < need; r++ { // a read yields ≥ 1 frame in practice
			n, err := rs.r.Read(rs.scratch)
			rs.pending = append(rs.pending, rs.scratch[:n]...)
			if err != nil {
				if out > 0 {
					return out, nil
				}
				return 0, err
			}
		}
		i := int(rs.pos)
		frac := rs.pos - float64(i)
		a := frame(rs.pending, i)
		b := frame(rs.pending, i+1)
		for ch := range 2 {
			v := float64(a[ch])*(1-frac) + float64(b[ch])*frac
			binary.LittleEndian.PutUint16(p[out:], uint16(int16(v)))
			out += 2
		}
		rs.pos += rs.step
		// Drop consumed source frames.
		if drop := int(rs.pos); drop > 0 {
			rs.pending = rs.pending[drop*4:]
			rs.pos -= float64(drop)
		}
	}
	return out, nil
}

func frame(b []byte, i int) [2]int16 {
	return [2]int16{int16(binary.LittleEndian.Uint16(b[i*4:])), int16(binary.LittleEndian.Uint16(b[i*4+2:]))}
}
