// Command video renders Prairie Loop's walkthrough: the 'Restore the Power' event in motion (fans, door,
// messages, alarm, teleport).
//
//	go run ./maps/prairieloop/video check     # camera never inside solid
//	go run ./maps/prairieloop/video render    # build/prairieloop/video/chunk_*.mp4 (2 workers)
//	go run ./maps/prairieloop/video audio     # build/prairieloop/video/audio.wav
//	go run ./maps/prairieloop/video stills T… # single-frame clips at the given times
//
// make video MAP=prairieloop runs check, render and audio, joins them and makes a Discord-size copy.
package main

import (
	"encoding/binary"
	"errors"
	"fmt"
	"math"
	"math/rand/v2"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"sync"

	"halflifemaps/maps/prairieloop/layout"
	"halflifemaps/tools/mapkit"
	"halflifemaps/tools/mapkit/paint"
	tx "halflifemaps/tools/mapkit/textures"
	"halflifemaps/tools/renderer"
)

const (
	fps, width, height = 30, 854, 480
	cut                = 57.0 // the teleport: the camera jumps from SEG_A to SEG_B
	tBreaker, tPower   = 5.0, 43.5
	tDoor0             = tPower + 1.5
	doorRise           = 104.0
	doorTime           = 104.0 / 110.0
	doorWait           = 14.0
	tAlarm0, tAlarm1   = tPower + 0.6, tPower + 16.0
	chunkFrames        = 220
)

type key struct {
	t           float64
	eye, target mapkit.V3
}

// segA: street -> breaker 3 -> transformer -> blue house -> reveal -> hall -> server room -> teleport pad.
var segA = []key{
	{0.0, mapkit.V3{-2600, 40, 95}, mapkit.V3{-2950, 200, 80}}, {2.5, mapkit.V3{-2780, 120, 85}, mapkit.V3{-2950, 200, 75}},
	{4.5, mapkit.V3{-2860, 150, 80}, mapkit.V3{-2944, 200, 70}}, {6.5, mapkit.V3{-2700, 60, 100}, mapkit.V3{-1500, 100, 100}},
	{10.0, mapkit.V3{-1700, 0, 110}, mapkit.V3{-900, 300, 80}}, {13.0, mapkit.V3{-1100, 320, 120}, mapkit.V3{-835, 410, 40}},
	{15.5, mapkit.V3{-700, 600, 110}, mapkit.V3{0, 640, 90}}, {18.0, mapkit.V3{250, 650, 80}, mapkit.V3{384, 900, 75}},
	{20.0, mapkit.V3{384, 760, 64}, mapkit.V3{384, 1000, 64}}, {21.3, mapkit.V3{370, 900, 64}, mapkit.V3{380, 1100, 64}},
	{22.5, mapkit.V3{380, 1080, 64}, mapkit.V3{600, 1150, 60}}, {23.7, mapkit.V3{570, 1190, 64}, mapkit.V3{300, 1224, -60}},
	{25.0, mapkit.V3{440, 1224, -60}, mapkit.V3{200, 1224, -150}}, {26.3, mapkit.V3{330, 1190, -116}, mapkit.V3{144, 1000, -140}},
	{27.6, mapkit.V3{300, 960, -116}, mapkit.V3{512, 880, -140}}, {28.8, mapkit.V3{512, 900, -116}, mapkit.V3{512, 700, -200}},
	{30.0, mapkit.V3{512, 760, -170}, mapkit.V3{512, 500, -260}}, {31.2, mapkit.V3{512, 640, -250}, mapkit.V3{512, 400, -265}},
	{32.4, mapkit.V3{512, 470, -258}, mapkit.V3{300, 0, -330}}, {34.0, mapkit.V3{512, 345, -258}, mapkit.V3{-100, -250, -340}},
	{36.0, mapkit.V3{500, 330, -256}, mapkit.V3{-400, -100, -380}}, {37.5, mapkit.V3{380, 336, -300}, mapkit.V3{0, 200, -400}},
	{39.0, mapkit.V3{220, 300, -398}, mapkit.V3{-320, 16, -110}}, {41.5, mapkit.V3{-100, 100, -380}, mapkit.V3{-560, -384, -400}},
	{43.0, mapkit.V3{-450, -150, -400}, mapkit.V3{-560, -384, -410}}, {46.5, mapkit.V3{-560, -300, -400}, mapkit.V3{-560, -560, -420}},
	{48.5, mapkit.V3{-560, -405, -400}, mapkit.V3{-560, -560, -405}}, {50.0, mapkit.V3{-590, -440, -400}, mapkit.V3{-690, -500, -420}},
	{51.5, mapkit.V3{-560, -420, -395}, mapkit.V3{-560, -250, -400}}, {52.5, mapkit.V3{-560, -320, -395}, mapkit.V3{-200, -200, -400}},
	{53.6, mapkit.V3{-250, -200, -390}, mapkit.V3{255, -330, -400}}, {55.5, mapkit.V3{180, -230, -390}, mapkit.V3{255, -330, -420}},
	{56.4, mapkit.V3{250, -300, -366}, mapkit.V3{200, 100, -380}}, {57.0, mapkit.V3{252, -310, -366}, mapkit.V3{150, 200, -380}},
}

// segB: arrive on the blue house roof -> roof loop -> steam vent -> wide shot.
var segB = []key{
	{57.0, mapkit.V3{384, 1150, 205}, mapkit.V3{384, 700, 150}}, {59.0, mapkit.V3{384, 1100, 205}, mapkit.V3{0, 0, 60}},
	{61.0, mapkit.V3{700, 1056, 205}, mapkit.V3{1344, 1056, 200}}, {64.0, mapkit.V3{1300, 1056, 205}, mapkit.V3{1344, 700, 200}},
	{66.0, mapkit.V3{1344, 760, 210}, mapkit.V3{1440, 576, 205}}, {68.0, mapkit.V3{1440, 500, 210}, mapkit.V3{1440, 0, 205}},
	{70.5, mapkit.V3{1440, 0, 210}, mapkit.V3{1440, -500, 205}}, {73.0, mapkit.V3{1440, -560, 210}, mapkit.V3{1344, -900, 205}},
	{75.0, mapkit.V3{1344, -900, 210}, mapkit.V3{1100, -1056, 205}}, {77.5, mapkit.V3{900, -1056, 210}, mapkit.V3{500, -1100, 190}},
	{80.0, mapkit.V3{600, -1250, 260}, mapkit.V3{384, -1316, 30}}, {82.5, mapkit.V3{430, -1400, 240}, mapkit.V3{384, -1310, 20}},
	{85.0, mapkit.V3{0, -900, 420}, mapkit.V3{0, 0, 0}}, {88.0, mapkit.V3{-300, -700, 560}, mapkit.V3{0, 100, 0}},
}

var dur = segB[len(segB)-1].t

func catmull(p0, p1, p2, p3 mapkit.V3, u float64) mapkit.V3 {
	var o mapkit.V3
	for i := range 3 {
		o[i] = 0.5 * ((2 * p1[i]) + (-p0[i]+p2[i])*u + (2*p0[i]-5*p1[i]+4*p2[i]-p3[i])*u*u + (-p0[i]+3*p1[i]-3*p2[i]+p3[i])*u*u*u)
	}
	return o
}

// cam is the camera's eye and target at time t: a Catmull-Rom path through the keys, eased a little.
func cam(t float64) (eye, target mapkit.V3) {
	k := segA
	if t >= cut {
		k = segB
	}
	t = min(max(t, k[0].t), k[len(k)-1].t-1e-6)
	i := 0
	for j := range len(k) - 1 {
		if k[j].t <= t {
			i = j
		}
	}
	u := (t - k[i].t) / (k[i+1].t - k[i].t)
	u = u*u*(3-2*u)*0.35 + u*0.65
	g := func(j int) key { return k[min(max(j, 0), len(k)-1)] }
	return catmull(g(i-1).eye, g(i).eye, g(i+1).eye, g(i+2).eye, u), catmull(g(i-1).target, g(i).target, g(i+1).target, g(i+2).target, u)
}

func doorH(t float64) float64 {
	switch {
	case t < tDoor0:
		return 0
	case t < tDoor0+doorTime:
		return doorRise * (t - tDoor0) / doorTime
	}
	t2 := t - (tDoor0 + doorTime + doorWait)
	if t2 < 0 {
		return doorRise
	}
	return max(0, doorRise-doorRise*t2/doorTime)
}

func fade(t, t0, hold float64) float64 {
	const fin, fout = 0.15, 0.6
	switch {
	case t < t0 || t > t0+fin+hold+fout:
		return 0
	case t < t0+fin:
		return (t - t0) / fin
	case t < t0+fin+hold:
		return 1
	}
	return 1 - (t-t0-fin-hold)/fout
}

func rz(a float64) [3][3]float64 {
	c, s := math.Cos(a), math.Sin(a)
	return [3][3]float64{{c, -s, 0}, {s, c, 0}, {0, 0, 1}}
}

// message is an on-screen game_text: its text layer and glow, drawn once and blended in while it shows.
type message struct {
	lay, glow *paint.Arr
}

func newMessage(msg string, y float64, c paint.Col) message {
	lay := paint.NewCanvas(width, height, paint.Col{})
	lay.Text(width/2, float64(int(height*y)), msg, c, tx.Bold(22), "mm")
	return message{lay.Arr(), lay.Blur(3).Arr()}
}

func (m message) draw(img []byte, alpha float64) {
	if alpha <= 0 {
		return
	}
	for i := range img {
		v := float64(img[i]) + (m.glow.P[i]*0.8+m.lay.P[i])*alpha
		img[i] = uint8(min(max(v, 0), 255))
	}
}

type paths struct{ bd, vid, bsp, valve string }

func getPaths() (paths, error) {
	bd, err := mapkit.BuildDir("prairieloop")
	if err != nil {
		return paths{}, err
	}
	p := paths{bd, filepath.Join(bd, "video"), filepath.Join(bd, "prairieloop.bsp"), filepath.Join(bd, "valve")}
	return p, os.MkdirAll(p.vid, 0o750)
}

func renderChunk(p paths, ci, f0, f1 int, out string) error {
	r, err := renderer.New(p.bsp, p.valve)
	if err != nil {
		return err
	}
	door := -1
	type fan struct {
		mi    int
		speed float64
	}
	var fans []fan
	for mi, kv := range r.MInfo {
		if kv["targetname"] == "server_door" {
			door = mi
		}
		if kv["classname"] == "func_rotating" {
			sp, _ := strconv.ParseFloat(kv["speed"], 64)
			fans = append(fans, fan{mi, sp})
		}
	}
	if door < 0 {
		return errNoDoor
	}
	breaker := newMessage("A breaker was switched on.  (Flip all 3 to restore power)", 0.7, paint.Col{255, 220, 90})
	power := newMessage("POWER RESTORED!  The JustinCon server room is open.", 0.25, paint.Col{255, 90, 200})
	tmp := out[:len(out)-4] + ".part.mp4"
	args := []string{"-y", "-loglevel", "error", "-f", "rawvideo", "-pix_fmt", "rgb24",
		"-s", fmt.Sprintf("%dx%d", width, height), "-r", strconv.Itoa(fps), "-i", "-", "-c:v", "libx264", "-crf", "19",
		"-preset", "veryfast", "-pix_fmt", "yuv420p", tmp}
	cmd := exec.Command("ffmpeg", args...) //nolint:gosec // a fixed program with this program's own arguments
	cmd.Stderr = os.Stderr
	in, err := cmd.StdinPipe()
	if err != nil {
		return err
	}
	if err := cmd.Start(); err != nil {
		return err
	}
	for fr := f0; fr < f1; fr++ {
		t := float64(fr) / fps
		eye, tgt := cam(t)
		dyn := map[int]renderer.Transform{door: {R: renderer.Identity, T: [3]float64{0, 0, doorH(t)}}}
		for _, f := range fans {
			dyn[f.mi] = renderer.Transform{R: rz(f.speed * t * math.Pi / 180), T: r.MOrigin[f.mi]}
		}
		img := r.Frame(eye, tgt, t, width, height, 80, dyn)
		breaker.draw(img, fade(t, tBreaker, 2.6))
		power.draw(img, fade(t, tPower, 4.5))
		if fl := max(0, 1-math.Abs(t-cut)/0.35); fl > 0 { // teleport flash
			for i, v := range img {
				img[i] = uint8(min(max(float64(v)*(1-fl)+255*fl, 0), 255))
			}
		}
		if _, err := in.Write(img); err != nil {
			return err
		}
		if fr%30 == 0 {
			fmt.Printf("chunk %d frame %d/%d\n", ci, fr-f0, f1-f0)
		}
	}
	if err := in.Close(); err != nil {
		return err
	}
	if err := cmd.Wait(); err != nil {
		return err
	}
	return os.Rename(tmp, out)
}

var errNoDoor = errors.New("no server_door in the map")

func render(p paths) error {
	nfr := int(dur * fps)
	type job struct {
		ci, f0, f1 int
		out        string
	}
	jobs := make(chan job)
	errs := make(chan error, 2)
	var wg sync.WaitGroup
	for range 2 {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for j := range jobs {
				if err := renderChunk(p, j.ci, j.f0, j.f1, j.out); err != nil {
					errs <- err
					return
				}
				fmt.Println("done", j.out)
			}
		}()
	}
	for i, s := 0, 0; s < nfr; i, s = i+1, s+chunkFrames {
		out := filepath.Join(p.vid, fmt.Sprintf("chunk_%02d.mp4", i))
		if _, err := os.Stat(out); err == nil {
			continue
		}
		jobs <- job{i, s, min(s+chunkFrames, nfr), out}
	}
	close(jobs)
	wg.Wait()
	close(errs)
	return <-errs
}

func readWAV(path string) ([]float64, error) {
	d, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	for i := 12; i+8 <= len(d); {
		id, n := string(d[i:i+4]), int(binary.LittleEndian.Uint32(d[i+4:]))
		if id == "data" {
			x := make([]float64, n/2)
			for k := range x {
				x[k] = float64(int16(binary.LittleEndian.Uint16(d[i+8+2*k:]))) / 32768 //nolint:gosec // 16-bit PCM is signed
			}
			return x, nil
		}
		i += 8 + n + n%2
	}
	return nil, fmt.Errorf("%w: %s", errNoData, path)
}

var errNoData = errors.New("no data chunk")

// interp is numpy.interp: linear between the points (xs ascending), clamped at the ends.
func interp(x float64, xs, ys []float64) float64 {
	if x <= xs[0] {
		return ys[0]
	}
	for i := 1; i < len(xs); i++ {
		if x <= xs[i] {
			return ys[i-1] + (ys[i]-ys[i-1])*(x-xs[i-1])/(xs[i]-xs[i-1])
		}
	}
	return ys[len(ys)-1]
}

// audio mixes the map's own loops the way the camera hears them: wind outdoors, fans down in the hall, the alarm
// while it sounds, and a low thump when the transformer blows.
func audio(p paths) error {
	const sr = 22050
	n := int(dur * sr)
	snd := filepath.Join(p.valve, "sound", "justincon")
	var loops [3][]float64
	for i, f := range []string{"wind_loop.wav", "fans_loop.wav", "alarm.wav"} {
		x, err := readWAV(filepath.Join(snd, f))
		if err != nil {
			return err
		}
		loops[i] = x
	}
	var houses [][4]float64
	for _, h := range layout.Houses {
		houses = append(houses, layout.Footprint(h))
	}
	var ts, gw, gf, ga []float64
	for k := 0; float64(k)*0.1 < dur+0.1; k++ {
		t := float64(k) * 0.1
		eye, _ := cam(t)
		z := eye[2]
		indoor := false
		for _, b := range houses {
			indoor = indoor || (b[0] <= eye[0] && eye[0] <= b[2] && b[1] <= eye[1] && eye[1] <= b[3] && 0 <= z && z <= 144)
		}
		w := 1.0
		if indoor {
			w = 0.45
		}
		ts = append(ts, t)
		gw = append(gw, w*min(max(1+min(z, 0)/140, 0), 1))
		f := min(max((-z-90)/170, 0), 1)
		if z < 0 && eye[1] > 850 {
			f += 0.25
		}
		gf = append(gf, f)
		a := 0.0
		if tAlarm0 <= t && t <= tAlarm1 {
			a = 0.15
			if z < -200 {
				a = 0.55
			}
		}
		ga = append(ga, a)
	}
	mix := make([]float64, n)
	for i := range mix {
		t := float64(i) / sr
		mix[i] = loops[0][i%len(loops[0])]*interp(t, ts, gw)*0.8 + loops[1][i%len(loops[1])]*interp(t, ts, gf)*0.9 +
			loops[2][i%len(loops[2])]*interp(t, ts, ga)
	}
	// distant transformer explosion: a low thump
	k0, l := int((tPower+0.2)*sr), int(1.2*sr)
	r := rand.New(rand.NewPCG(3, 0)) //nolint:gosec // sound design, not security
	burst := make([]float64, l)
	for i := range burst {
		burst[i] = r.NormFloat64() * math.Exp(-float64(i)/(0.25*sr))
	}
	for i := range l { // moving average over 60 samples (numpy's convolve "same"), times 2.5
		s := 0.0
		for j := i - 30; j < i+30; j++ {
			if j >= 0 && j < l {
				s += burst[j]
			}
		}
		if k0+i < n {
			mix[k0+i] += s / 60 * 2.5 * 0.5
		}
	}
	return mapkit.WriteWAV(filepath.Join(p.vid, "audio.wav"), mix, sr)
}

func check(p paths) error {
	r, err := renderer.New(p.bsp, p.valve)
	if err != nil {
		return err
	}
	var bad []float64
	for f := range int(dur * fps) {
		t := float64(f) / fps
		if eye, _ := cam(t); r.Solid(eye) {
			bad = append(bad, math.Round(t*100)/100)
		}
	}
	fmt.Println("camera-in-solid frames:", len(bad), bad[:min(len(bad), 40)])
	return nil
}

func run(args []string) error {
	if len(args) == 0 {
		return errUsage
	}
	p, err := getPaths()
	if err != nil {
		return err
	}
	switch args[0] {
	case "check":
		return check(p)
	case "render":
		return render(p)
	case "audio":
		return audio(p)
	case "stills":
		for _, a := range args[1:] {
			t, err := strconv.ParseFloat(a, 64)
			if err != nil {
				return err
			}
			f := int(t * fps)
			if err := renderChunk(p, 0, f, f+1, filepath.Join(p.vid, fmt.Sprintf("still_%05.1f.mp4", t))); err != nil {
				return err
			}
		}
		return nil
	}
	return errUsage
}

var errUsage = errors.New("usage: video check|render|audio|stills T")

func main() {
	if err := run(os.Args[1:]); err != nil {
		fmt.Fprintln(os.Stderr, "video:", err)
		os.Exit(1)
	}
}
