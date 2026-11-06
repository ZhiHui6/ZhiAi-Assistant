package winstate

import "time"

type State string

const (
	StateHidden     State = "hidden"
	StateEdgeIdle   State = "edge-idle"
	StateEdgeOpen   State = "edge-open"
	StateManualIdle State = "manual-idle"
	StateFloating   State = "floating"
)

type Edge string

const (
	EdgeRight Edge = "right"
	EdgeLeft  Edge = "left"
)

type Point struct{ X, Y int }

type Rect struct{ X, Y, W, H int }

func (r Rect) Right() int  { return r.X + r.W }
func (r Rect) Bottom() int { return r.Y + r.H }

func (r Rect) Contains(p Point) bool {
	return p.X >= r.X && p.X < r.Right() && p.Y >= r.Y && p.Y < r.Bottom()
}

type Monitor struct {
	Index   int
	Rect    Rect
	Work    Rect
	Dpi     int
	Primary bool
}

type Placement struct {
	X, Y, W, H int
	NoActivate bool
	Visible    bool
}

type Controller interface {
	Apply(p Placement)
	ActivateWindow()
	IsForeground() bool
}

type Config struct {
	Edge        Edge
	PanelW      int
	Peek        int
	HandleW     int
	EdgeEnabled bool
	AnimDur     time.Duration
	CloseHold   time.Duration
	Monitors    []Monitor
	TargetIndex int
}

func DefaultConfig() Config {
	return Config{
		Edge:        EdgeRight,
		PanelW:      340,
		Peek:        30,
		HandleW:     30,
		EdgeEnabled: false,
		AnimDur:     0,
		CloseHold:   180 * time.Millisecond,
	}
}

type Machine struct {
	ctrl Controller
	cfg  Config
	mon  Monitor

	state     State
	pinned    bool
	activated bool

	placement Placement
	x         int

	animFrom  int
	animTo    int
	animStart time.Time
	animating bool

	closeAt time.Time
	closeX  int

	cursor Point

	onChange []func(State)
}

func New(ctrl Controller, cfg Config) *Machine {
	m := &Machine{ctrl: ctrl, cfg: cfg, state: StateHidden}
	m.refreshMonitor()
	m.x = m.idleX()
	m.placement = m.edgePlacement(m.x)
	m.placement.NoActivate = true
	return m
}

func (m *Machine) State() State         { return m.state }
func (m *Machine) Placement() Placement { return m.placement }
func (m *Machine) Pinned() bool         { return m.pinned }
func (m *Machine) Config() Config       { return m.cfg }
func (m *Machine) Animating() bool      { return m.animating }
func (m *Machine) LastCursor() Point    { return m.cursor }

func (m *Machine) OnChange(fn func(State)) { m.onChange = append(m.onChange, fn) }

func (m *Machine) setState(s State) {
	if m.state == s {
		return
	}
	m.state = s
	for _, fn := range m.onChange {
		fn(s)
	}
}

func (m *Machine) refreshMonitor() {
	m.mon = pickMonitor(m.cfg.Monitors, m.cfg.TargetIndex)
	if m.mon.Work.W <= 0 {
		m.mon = Monitor{
			Rect:    Rect{0, 0, 1920, 1080},
			Work:    Rect{0, 0, 1920, 1040},
			Dpi:     96,
			Primary: true,
		}
	}
}

func pickMonitor(mons []Monitor, idx int) Monitor {
	if len(mons) == 0 {
		return Monitor{}
	}
	for _, mo := range mons {
		if mo.Index == idx {
			return mo
		}
	}
	for _, mo := range mons {
		if mo.Primary {
			return mo
		}
	}
	return mons[0]
}

func (m *Machine) openX() int {
	if m.cfg.Edge == EdgeLeft {
		return m.mon.Work.X
	}
	return m.mon.Work.Right() - m.cfg.PanelW
}

func (m *Machine) idleX() int {
	if m.cfg.Edge == EdgeLeft {
		return m.mon.Work.X - m.cfg.PanelW + m.cfg.Peek
	}
	return m.mon.Work.Right() - m.cfg.Peek
}

func (m *Machine) manualIdleX() int {
	if m.cfg.Edge == EdgeLeft {
		return m.mon.Work.X - m.cfg.PanelW + m.cfg.HandleW
	}
	return m.mon.Work.Right() - m.cfg.HandleW
}

func (m *Machine) noActivate() bool {
	if m.state == StateManualIdle {
		return false
	}
	return !m.activated
}

func (m *Machine) edgePlacement(x int) Placement {
	return Placement{X: x, Y: m.mon.Work.Y, W: m.cfg.PanelW, H: m.mon.Work.H}
}

func (m *Machine) docked() bool { return !m.cfg.EdgeEnabled }

func (m *Machine) floatingBase() Placement {
	return Placement{X: m.openX(), Y: m.mon.Work.Y, W: m.cfg.PanelW, H: m.mon.Work.H}
}

func (m *Machine) inEdgeBand(p Point) bool {
	if !m.cfg.EdgeEnabled {
		return false
	}
	band := m.cfg.Peek
	if band < 2 {
		band = 2
	}
	if m.cfg.Edge == EdgeLeft {
		return p.X >= m.mon.Work.X-band && p.X <= m.mon.Work.X+band &&
			p.Y >= m.mon.Work.Y && p.Y < m.mon.Work.Bottom()
	}
	return p.X >= m.mon.Work.Right()-band && p.X <= m.mon.Work.Right()+band &&
		p.Y >= m.mon.Work.Y && p.Y < m.mon.Work.Bottom()
}

func (m *Machine) basePlacement() Placement {
	if m.state == StateFloating {
		return m.floatingBase()
	}
	return m.edgePlacement(m.x)
}

func (m *Machine) cancelClose() { m.closeAt = time.Time{} }

func (m *Machine) beginClose(target int, now time.Time) {
	m.animating = false
	if m.cfg.CloseHold <= 0 {
		m.x = target
		p := m.edgePlacement(m.x)
		p.NoActivate = m.noActivate()
		p.Visible = true
		m.placement = p
		m.ctrl.Apply(p)
		m.closeAt = time.Time{}
		return
	}
	m.closeAt = now.Add(m.cfg.CloseHold)
	m.closeX = target
}

func (m *Machine) moveTo(target int, now time.Time) {
	p := m.basePlacement()
	p.NoActivate = m.noActivate()
	p.Visible = true
	if m.cfg.AnimDur <= 0 || target == m.x {
		m.x = target
		m.animating = false
		p.X = target
		m.placement = p
		m.ctrl.Apply(p)
		return
	}
	m.animFrom = m.x
	m.animTo = target
	m.animStart = now
	m.animating = true
	p.X = m.x
	m.placement = p
	m.ctrl.Apply(p)
}

func (m *Machine) Reveal(now time.Time) {
	m.cancelClose()
	m.refreshMonitor()
	m.activated = true
	if m.docked() {
		m.setState(StateEdgeOpen)
		m.moveTo(m.openX(), now)
		p := m.placement
		p.NoActivate = false
		m.placement = p
		m.ctrl.Apply(p)
		m.ctrl.ActivateWindow()
		return
	}
	m.setState(StateFloating)
	m.moveTo(m.openX(), now)
	p := m.placement
	p.NoActivate = false
	m.placement = p
	m.ctrl.Apply(p)
	m.ctrl.ActivateWindow()
}

func (m *Machine) OpenByEdge(now time.Time) {
	if m.state == StateEdgeOpen || m.state == StateFloating {
		return
	}
	m.cancelClose()
	m.refreshMonitor()
	m.setState(StateEdgeOpen)
	m.moveTo(m.openX(), now)
}

func (m *Machine) Collapse(now time.Time) {
	m.activated = false
	if m.docked() {
		if m.state == StateManualIdle {
			return
		}
		m.setState(StateManualIdle)
		m.beginClose(m.manualIdleX(), now)
		return
	}
	if m.state != StateEdgeIdle {
		m.setState(StateEdgeIdle)
	}
	m.beginClose(m.idleX(), now)
}

func (m *Machine) Hide() {
	m.cancelClose()
	m.animating = false
	m.activated = false
	p := m.placement
	p.Visible = false
	p.NoActivate = true
	m.placement = p
	m.ctrl.Apply(p)
	m.setState(StateHidden)
}

func (m *Machine) ShowIdle(now time.Time) {
	m.cancelClose()
	m.refreshMonitor()
	m.activated = false
	m.animating = false
	if m.docked() {
		m.x = m.openX()
		p := m.edgePlacement(m.x)
		p.NoActivate = true
		p.Visible = true
		m.placement = p
		m.ctrl.Apply(p)
		m.setState(StateEdgeOpen)
		return
	}
	m.x = m.idleX()
	p := m.edgePlacement(m.x)
	p.NoActivate = true
	p.Visible = true
	m.placement = p
	m.ctrl.Apply(p)
	m.setState(StateEdgeIdle)
}

func (m *Machine) ToggleFloat(now time.Time) {
	if m.docked() {
		if m.state == StateHidden || m.state == StateManualIdle {
			m.ShowIdle(now)
			return
		}
		m.Collapse(now)
		return
	}
	if m.state == StateFloating {
		m.Collapse(now)
		return
	}
	m.Reveal(now)
}

func (m *Machine) SetPinned(v bool) {
	m.pinned = v
	if !v {
	}
}

func (m *Machine) Activate() {
	m.activated = true
	p := m.placement
	p.NoActivate = false
	m.placement = p
	m.ctrl.Apply(p)
	m.ctrl.ActivateWindow()
}

func (m *Machine) rect() Rect {
	return Rect{X: m.placement.X, Y: m.placement.Y, W: m.placement.W, H: m.placement.H}
}

func (m *Machine) OnCursor(p Point, now time.Time, foreground bool) {
	m.cursor = p
	switch m.state {
	case StateEdgeIdle:
		if !m.closeAt.IsZero() && m.rect().Contains(p) {
			m.cancelClose()
			if m.placement.H != m.mon.Work.H {
				m.setState(StateFloating)
			} else {
				m.setState(StateEdgeOpen)
			}
			return
		}
		if m.inEdgeBand(p) {
			m.OpenByEdge(now)
		}
	case StateEdgeOpen:
		if m.docked() {
			return
		}
		if m.placement.Visible && !m.rect().Contains(p) && !m.pinned {
			m.Collapse(now)
		}
	case StateFloating:
		if !foreground && m.activated && !m.pinned {
			m.Collapse(now)
		}
	}
}

func (m *Machine) Tick(now time.Time, foreground bool) bool {
	if !m.closeAt.IsZero() {
		if now.Before(m.closeAt) {
			return true
		}
		m.x = m.closeX
		p := m.edgePlacement(m.x)
		p.NoActivate = m.noActivate()
		p.Visible = true
		m.placement = p
		m.ctrl.Apply(p)
		m.closeAt = time.Time{}
	}
	if m.state == StateFloating && !foreground && m.activated && !m.pinned {
		m.Collapse(now)
	}
	if !m.animating {
		return false
	}
	dur := m.cfg.AnimDur
	if dur <= 0 {
		dur = time.Millisecond
	}
	el := now.Sub(m.animStart)
	if el >= dur {
		m.x = m.animTo
		m.animating = false
	} else {
		t := float64(el) / float64(dur)
		ease := t * t * (3 - 2*t)
		m.x = m.animFrom + int(float64(m.animTo-m.animFrom)*ease)
	}
	p := m.basePlacement()
	p.X = m.x
	p.Visible = true
	p.NoActivate = m.noActivate()
	m.placement = p
	m.ctrl.Apply(p)
	return m.animating || !m.closeAt.IsZero()
}

func (m *Machine) SetConfig(cfg Config) {
	m.cfg = cfg
	m.cancelClose()
	m.refreshMonitor()
	if m.state == StateHidden {
		if m.docked() {
			m.ShowIdle(time.Now())
		}
		return
	}
	m.animating = false
	if m.state == StateManualIdle {
		if m.docked() {
			m.x = m.manualIdleX()
			p := m.edgePlacement(m.x)
			p.NoActivate = false
			p.Visible = true
			m.placement = p
			m.ctrl.Apply(p)
			return
		}
		m.x = m.idleX()
		p := m.edgePlacement(m.x)
		p.NoActivate = true
		p.Visible = true
		m.placement = p
		m.ctrl.Apply(p)
		m.setState(StateEdgeIdle)
		return
	}
	switch m.state {
	case StateFloating:
		m.activated = true
		m.x = m.openX()
		p := m.floatingBase()
		p.NoActivate = false
		p.Visible = true
		m.placement = p
		m.ctrl.Apply(p)
	case StateEdgeOpen:
		m.x = m.openX()
		p := m.edgePlacement(m.x)
		p.NoActivate = !m.activated
		p.Visible = true
		m.placement = p
		m.ctrl.Apply(p)
	default:
		if m.docked() {
			m.x = m.openX()
			p := m.edgePlacement(m.x)
			p.NoActivate = true
			p.Visible = true
			m.placement = p
			m.ctrl.Apply(p)
			m.setState(StateEdgeOpen)
			return
		}
		m.x = m.idleX()
		p := m.edgePlacement(m.x)
		p.NoActivate = true
		p.Visible = true
		m.placement = p
		m.ctrl.Apply(p)
	}
}

func (m *Machine) OnDisplayChange(now time.Time) {
	switch m.state {
	case StateHidden:
		return
	case StateFloating:
		m.refreshMonitor()
		m.SetConfig(m.cfg)
	case StateManualIdle:
		m.refreshMonitor()
		m.SetConfig(m.cfg)
	case StateEdgeOpen:
		m.refreshMonitor()
		m.SetConfig(m.cfg)
	default:
		m.refreshMonitor()
		m.ShowIdle(now)
	}
}
