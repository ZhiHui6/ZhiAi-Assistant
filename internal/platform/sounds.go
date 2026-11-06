package platform

import (
	"encoding/binary"
	"math"
	"os"
	"path/filepath"

	"zhi-ai-assistant/internal/core/store"
)

const soundSampleRate = 44100

const soundVersion = "v1"

type soundNote struct {
	start  float64
	dur    float64
	freq   float64
	toFreq float64
	amp    float64
	attack float64
	decay  float64
	harm   [][2]float64
}

type soundSpec struct {
	length float64
	notes  []soundNote
}

var soundSpecs = map[string]soundSpec{
	"chime": {length: 1.35, notes: []soundNote{
		{start: 0, dur: 0.85, freq: 1318.51, amp: 0.55, attack: 0.004, decay: 4.2, harm: [][2]float64{{1, 1}, {2, 0.34}, {3, 0.1}}},
		{start: 0.11, dur: 1.15, freq: 1975.53, amp: 0.42, attack: 0.004, decay: 3.4, harm: [][2]float64{{1, 1}, {2, 0.3}}},
	}},
	"bell": {length: 2.3, notes: []soundNote{
		{start: 0, dur: 2.0, freq: 523.25, amp: 0.5, attack: 0.003, decay: 1.5, harm: [][2]float64{{1, 1}, {2, 0.45}, {2.76, 0.22}, {5.4, 0.1}}},
		{start: 0, dur: 2.2, freq: 261.63, amp: 0.3, attack: 0.006, decay: 1.2, harm: [][2]float64{{1, 1}, {2, 0.2}}},
	}},
	"marimba": {length: 1.25, notes: []soundNote{
		{start: 0, dur: 0.55, freq: 987.77, amp: 0.55, attack: 0.002, decay: 9, harm: [][2]float64{{1, 1}, {4, 0.22}, {7, 0.06}}},
		{start: 0.14, dur: 0.6, freq: 1318.51, amp: 0.45, attack: 0.002, decay: 8, harm: [][2]float64{{1, 1}, {4, 0.2}}},
		{start: 0.28, dur: 0.9, freq: 1975.53, amp: 0.4, attack: 0.002, decay: 6, harm: [][2]float64{{1, 1}, {4, 0.16}}},
	}},
	"arpeggio": {length: 1.2, notes: []soundNote{
		{start: 0, dur: 0.5, freq: 523.25, amp: 0.5, attack: 0.004, decay: 5, harm: [][2]float64{{1, 1}, {2, 0.25}}},
		{start: 0.16, dur: 0.55, freq: 659.25, amp: 0.48, attack: 0.004, decay: 5, harm: [][2]float64{{1, 1}, {2, 0.24}}},
		{start: 0.32, dur: 0.8, freq: 783.99, amp: 0.46, attack: 0.004, decay: 4, harm: [][2]float64{{1, 1}, {2, 0.22}}},
	}},
	"dawn": {length: 2.6, notes: []soundNote{
		{start: 0, dur: 2.4, freq: 523.25, amp: 0.34, attack: 0.35, decay: 0.9, harm: [][2]float64{{1, 1}, {3, 0.18}, {5, 0.06}}},
		{start: 0.02, dur: 2.4, freq: 659.25, amp: 0.3, attack: 0.38, decay: 0.9, harm: [][2]float64{{1, 1}, {3, 0.16}}},
		{start: 0.04, dur: 2.4, freq: 783.99, amp: 0.26, attack: 0.42, decay: 0.9, harm: [][2]float64{{1, 1}, {3, 0.14}}},
	}},
	"pulse": {length: 0.95, notes: []soundNote{
		{start: 0, dur: 0.07, freq: 880, amp: 0.5, attack: 0.002, decay: 18, harm: [][2]float64{{1, 1}, {3, 0.2}}},
		{start: 0.18, dur: 0.07, freq: 880, amp: 0.5, attack: 0.002, decay: 18, harm: [][2]float64{{1, 1}, {3, 0.2}}},
		{start: 0.36, dur: 0.07, freq: 880, amp: 0.5, attack: 0.002, decay: 18, harm: [][2]float64{{1, 1}, {3, 0.2}}},
		{start: 0.54, dur: 0.07, freq: 1174.66, amp: 0.45, attack: 0.002, decay: 16, harm: [][2]float64{{1, 1}, {3, 0.18}}},
	}},
	"drop": {length: 1.0, notes: []soundNote{
		{start: 0, dur: 0.5, freq: 1200, toFreq: 520, amp: 0.6, attack: 0.002, decay: 7, harm: [][2]float64{{1, 1}, {2, 0.15}}},
		{start: 0.3, dur: 0.4, freq: 760, toFreq: 460, amp: 0.18, attack: 0.002, decay: 9, harm: [][2]float64{{1, 1}}},
	}},
	"chord": {length: 2.3, notes: []soundNote{
		{start: 0, dur: 2.0, freq: 440, amp: 0.4, attack: 0.06, decay: 1.4, harm: [][2]float64{{1, 1}, {2, 0.3}, {3, 0.12}}},
		{start: 0.01, dur: 2.1, freq: 659.25, amp: 0.34, attack: 0.07, decay: 1.3, harm: [][2]float64{{1, 1}, {2, 0.26}}},
		{start: 0.02, dur: 2.2, freq: 880, amp: 0.3, attack: 0.08, decay: 1.2, harm: [][2]float64{{1, 1}, {2, 0.22}}},
	}},
	"wood": {length: 0.45, notes: []soundNote{
		{start: 0, dur: 0.09, freq: 1500, amp: 0.6, attack: 0.001, decay: 26, harm: [][2]float64{{1, 1}, {1.7, 0.3}}},
		{start: 0.11, dur: 0.1, freq: 1100, amp: 0.5, attack: 0.001, decay: 24, harm: [][2]float64{{1, 1}, {1.6, 0.25}}},
	}},
	"dingdong": {length: 1.7, notes: []soundNote{
		{start: 0, dur: 0.9, freq: 659.25, amp: 0.55, attack: 0.003, decay: 3.4, harm: [][2]float64{{1, 1}, {2.4, 0.28}, {4, 0.08}}},
		{start: 0.4, dur: 1.2, freq: 523.25, amp: 0.5, attack: 0.003, decay: 3.0, harm: [][2]float64{{1, 1}, {2.4, 0.26}}},
	}},
}

var legacySoundAlias = map[string]string{
	"exclaim":  "chime",
	"asterisk": "dingdong",
	"hand":     "pulse",
	"question": "arpeggio",
	"ok":       "chord",
}

func hasBuiltinSound(id string) bool {
	_, ok := soundSpecs[id]
	return ok
}

func renderSound(spec soundSpec) []float64 {
	total := int(math.Ceil(spec.length * soundSampleRate))
	buf := make([]float64, total)
	for _, nt := range spec.notes {
		offset := int(nt.start * soundSampleRate)
		count := int(nt.dur * soundSampleRate)
		for i := 0; i < count; i++ {
			idx := offset + i
			if idx >= total {
				break
			}
			t := float64(i) / soundSampleRate
			env := 1.0
			if nt.attack > 0 && t < nt.attack {
				env = t / nt.attack
			}
			env *= math.Exp(-nt.decay * t)
			freq := nt.freq
			if nt.toFreq > 0 && nt.dur > 0 {
				freq = nt.freq + (nt.toFreq-nt.freq)*(t/nt.dur)
			}
			var s float64
			if len(nt.harm) == 0 {
				s = math.Sin(2 * math.Pi * freq * t)
			} else {
				for _, h := range nt.harm {
					s += h[1] * math.Sin(2*math.Pi*freq*h[0]*t)
				}
			}
			buf[idx] += s * env * nt.amp
		}
	}
	return buf
}

func encodeWAV(samples []float64) []byte {
	peak := 0.0
	for _, s := range samples {
		if a := math.Abs(s); a > peak {
			peak = a
		}
	}
	scale := 1.0
	if peak > 0 {
		scale = 0.89 / peak
	}
	dataLen := len(samples) * 2
	buf := make([]byte, 44+dataLen)
	copy(buf[0:], "RIFF")
	binary.LittleEndian.PutUint32(buf[4:], uint32(36+dataLen))
	copy(buf[8:], "WAVE")
	copy(buf[12:], "fmt ")
	binary.LittleEndian.PutUint32(buf[16:], 16)
	binary.LittleEndian.PutUint16(buf[20:], 1)
	binary.LittleEndian.PutUint16(buf[22:], 1)
	binary.LittleEndian.PutUint32(buf[24:], soundSampleRate)
	binary.LittleEndian.PutUint32(buf[28:], soundSampleRate*2)
	binary.LittleEndian.PutUint16(buf[32:], 2)
	binary.LittleEndian.PutUint16(buf[34:], 16)
	copy(buf[36:], "data")
	binary.LittleEndian.PutUint32(buf[40:], uint32(dataLen))
	for i, s := range samples {
		v := s * scale
		if v > 1 {
			v = 1
		} else if v < -1 {
			v = -1
		}
		binary.LittleEndian.PutUint16(buf[44+i*2:], uint16(int16(math.Round(v*32767))))
	}
	return buf
}

func ensureBuiltinSound(id string) (string, error) {
	spec, ok := soundSpecs[id]
	if !ok {
		return "", os.ErrNotExist
	}
	dir := filepath.Join(store.DataDir(), "cache", "sounds")
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return "", err
	}
	path := filepath.Join(dir, "zai-"+soundVersion+"-"+id+".wav")
	if st, err := os.Stat(path); err == nil && st.Size() > 44 {
		return path, nil
	}
	if err := os.WriteFile(path, encodeWAV(renderSound(spec)), 0o644); err != nil {
		return "", err
	}
	return path, nil
}
