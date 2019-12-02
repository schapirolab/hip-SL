// Copyright (c) 2019, The Emergent Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

// Adapted from the github.com/emer/leabra/hip hippocampus example for statistical learning
package main

import (
	"encoding/csv"
	"flag"
	"fmt"
	"io/ioutil"
	"log"
	"math/rand"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/emer/emergent/emer"
	"github.com/emer/emergent/env"
	"github.com/emer/emergent/netview"
	"github.com/emer/emergent/params"
	"github.com/emer/emergent/prjn"
	"github.com/emer/emergent/relpos"

	"github.com/emer/etable/agg"
	"github.com/emer/etable/eplot"
	"github.com/emer/etable/etable"
	"github.com/emer/etable/etensor"
	_ "github.com/emer/etable/etview" // include to get gui views
	"github.com/emer/etable/split"

	"github.com/emer/leabra/hip"
	"github.com/emer/leabra/leabra"

	"github.com/goki/gi/gi"
	"github.com/goki/gi/gimain"
	"github.com/goki/gi/giv"
	"github.com/goki/ki/ki"
	"github.com/goki/ki/kit"
)

// this is the stub main for gogi that calls our actual mainrun function, at end of file
func main() {
	gimain.Main(func() {
		mainrun()
	})
}

// LogPrec is precision for saving float values in logs
const LogPrec = 4

// ParamSets is the default set of parameters -- Base is always applied, and others can be optionally
// selected to apply on top of that
var ParamSets = params.Sets{
	{Name: "Base", Desc: "these are the best params", Sheets: params.Sheets{
		"Network": &params.Sheet{
			{Sel: "Prjn", Desc: "keeping default params for generic prjns",
				Params: params.Params{
					"Prjn.Learn.Momentum.On": "true",
					"Prjn.Learn.Norm.On":     "true",
					"Prjn.Learn.WtBal.On":    "false",
				}},
			{Sel: ".EcCa1Prjn", Desc: "encoder projections -- no norm, moment",
				Params: params.Params{
					"Prjn.Learn.Lrate":       "0.02",
					"Prjn.Learn.Momentum.On": "false",
					"Prjn.Learn.Norm.On":     "false",
					"Prjn.Learn.WtBal.On":    "false",
				}},
			{Sel: ".HippoCHL", Desc: "hippo CHL projections -- no norm, moment, but YES wtbal = sig better",
				Params: params.Params{
					"Prjn.CHL.Hebb":          "0.05",
					"Prjn.Learn.Lrate":       "0.4",
					"Prjn.Learn.Momentum.On": "false",
					"Prjn.Learn.Norm.On":     "false",
					"Prjn.Learn.WtBal.On":    "true",
				}},
			{Sel: "#CA1ToECout", Desc: "extra strong from CA1 to ECout",
				Params: params.Params{
					"Prjn.WtScale.Abs": "1.0",
					"Prjn.WtScale.Rel": "1.0",
					"Prjn.Learn.Lrate": "0.02",
					"Prjn.WtInit.Mean": "0.5",
					"Prjn.WtInit.Var":  "0.25",
				}},
			{Sel: "#InputToECin", Desc: "one-to-one input to EC",
				Params: params.Params{
					"Prjn.Learn.Learn": "false",
					"Prjn.WtInit.Mean": "0.5",
					"Prjn.WtInit.Var":  "0.25",
				}},
			{Sel: "#ECoutToECin", Desc: "one-to-one out to in",
				Params: params.Params{
					"Prjn.Learn.Learn": "false",
					"Prjn.WtInit.Mean": "0.5",
					"Prjn.WtInit.Var":  "0.01",
					"Prjn.WtScale.Abs": "2.0",
					"Prjn.WtScale.Rel": "0.5",
				}},
			{Sel: "#DGToCA3", Desc: "Mossy fibers: strong, non-learning",
				Params: params.Params{
					"Prjn.CHL.Hebb":    "0.001",
					"Prjn.CHL.SAvgCor": "1",
					"Prjn.Learn.Learn": "false",
					"Prjn.WtInit.Mean": "0.9",
					"Prjn.WtInit.Var":  "0.01",
					"Prjn.WtScale.Abs": "1.0", //
					"Prjn.WtScale.Rel": "8",
				}},
			{Sel: "#CA3ToCA3", Desc: "CA3 recurrent cons",
				Params: params.Params{
					"Prjn.CHL.Hebb":    "0.01",
					"Prjn.CHL.SAvgCor": "1",
					"Prjn.WtScale.Rel": "2",
					"Prjn.WtInit.Mean": "0.5",
					"Prjn.WtInit.Var":  "0.25",
				}},
			{Sel: "#CA3ToCA1", Desc: "Schaffer collaterals -- slower, less hebb",
				Params: params.Params{
					"Prjn.CHL.Hebb":    "0.005",
					"Prjn.CHL.SAvgCor": "0.4",
					"Prjn.Learn.Lrate": "0.05",
					"Prjn.WtInit.Mean": "0.5",
					"Prjn.WtInit.Var":  "0.25",
					"Prjn.WtScale.Abs": "1.0",
					"Prjn.WtScale.Rel": "1.0",
				}},
			{Sel: ".EC", Desc: "all EC layers: only pools, no layer-level",
				Params: params.Params{
					"Layer.Act.Gbar.L":        ".1",
					"Layer.Inhib.ActAvg.Init": "0.2",
					"Layer.Inhib.Layer.On":    "true",
					"Layer.Inhib.Layer.Gi":    "2",
				}},
			{Sel: "#DG", Desc: "very sparse = high inibhition",
				Params: params.Params{
					"Layer.Inhib.ActAvg.Init": "0.01",
					"Layer.Inhib.Layer.Gi":    "3.6",
				}},
			{Sel: "#CA3", Desc: "sparse = high inibhition",
				Params: params.Params{
					"Layer.Inhib.ActAvg.Init": "0.02",
					"Layer.Inhib.Layer.Gi":    "2.8",
				}},
			{Sel: "#CA1", Desc: "CA1 only Pools",
				Params: params.Params{
					"Layer.Inhib.ActAvg.Init": "0.1",
					"Layer.Inhib.Layer.On":    "true",
					"Layer.Inhib.Layer.Gi":    "2",
				}},
		},
		"Sim": &params.Sheet{
			{Sel: "Sim", Desc: "best params always finish in this time",
				Params: params.Params{
					"Sim.MaxEpcs": "10",
				}},
		},
	}},
}

// Sim encapsulates the entire simulation model, and we define all the
// functionality as methods on this struct.  This structure keeps all relevant
// state information organized and available without having to pass everything around
// as arguments to methods, and provides the core GUI interface (note the view tags
// for the fields which provide hints to how things should be displayed).
type Sim struct {
	Net           *leabra.Network   `view:"no-inline"`
	TrainSL       *etable.Table     `view:"no-inline" desc:"SL training patterns to use"`
	TrainEpisodic *etable.Table     `view:"no-inline" desc:"Episodic training patterns to use"`
	TestSL        *etable.Table     `view:"no-inline" desc:"SL testing patterns to use"`
	TrnTrlLog     *etable.Table     `view:"no-inline" desc:"training trial-level log data"`
	TrnEpcLog     *etable.Table     `view:"no-inline" desc:"training epoch-level log data"`
	TstEpcLog     *etable.Table     `view:"no-inline" desc:"testing epoch-level log data"`
	TstTrlLog     *etable.Table     `view:"no-inline" desc:"testing trial-level log data"`
	TstErrLog     *etable.Table     `view:"no-inline" desc:"log of all test trials where errors were made"`
	TstErrStats   *etable.Table     `view:"no-inline" desc:"stats on test trials where errors were made"`
	TstCycLog     *etable.Table     `view:"no-inline" desc:"testing cycle-level log data"`
	RunLog        *etable.Table     `view:"no-inline" desc:"summary log of each run"`
	RunStats      *etable.Table     `view:"no-inline" desc:"aggregate stats on all runs"`
	Params        params.Sets       `view:"no-inline" desc:"full collection of param sets"`
	ParamSet      string            `desc:"which set of *additional* parameters to use -- always applies Base and optionaly this next if set"`
	Tag           string            `desc:"extra tag string to add to any file names output from sim (e.g., weights files, log files, params)"`
	MaxRuns       int               `desc:"maximum number of model runs to perform"`
	MaxEpcs       int               `desc:"maximum number of epochs to run per model run"`
	TrialperEpc   int               `desc:"number of trials per epoch of training"`
	TrainEnv      env.FixedTable    `desc:"Training environment -- contains everything about iterating over input / output patterns over training"`
	TestEnv       env.FixedTable    `desc:"Testing environment -- manages iterating over testing"`
	Time          leabra.Time       `desc:"leabra timing parameters and state"`
	ViewOn        bool              `desc:"whether to update the network view while running"`
	TrainUpdt     leabra.TimeScales `desc:"at what time scale to update the display during training?  Anything longer than Epoch updates at Epoch in this model"`
	TestUpdt      leabra.TimeScales `desc:"at what time scale to update the display during testing?  Anything longer than Epoch updates at Epoch in this model"`
	TestInterval  int               `desc:"how often to run through all the test patterns, in terms of training epochs"`
	LayStatNms    []string          `desc:"names of layers to collect more detailed stats on (avg act, etc)"`
	MemThr        float64           `desc:"threshold to use for memory test -- if error proportion is below this number, it is scored as a correct trial"`

	// statistics: note use float64 as that is best for etable.Table
	TestNm         string  `inactive:"+" desc:"what set of patterns are we currently testing"`
	Mem            float64 `inactive:"+" desc:"whether current trial's ECout met memory criterion"`
	TrgOnWasOffAll float64 `inactive:"+" desc:"current trial's proportion of bits where target = on but ECout was off ( < 0.5), for all bits"`
	TrgOnWasOffCmp float64 `inactive:"+" desc:"current trial's proportion of bits where target = on but ECout was off ( < 0.5), for only completion bits that were not active in ECin"`
	TrgOffWasOn    float64 `inactive:"+" desc:"current trial's proportion of bits where target = off but ECout was on ( > 0.5)"`
	TrlSSE         float64 `inactive:"+" desc:"current trial's sum squared error"`
	TrlAvgSSE      float64 `inactive:"+" desc:"current trial's average sum squared error"`
	TrlCosDiff     float64 `inactive:"+" desc:"current trial's cosine difference"`

	EpcSSE     float64 `inactive:"+" desc:"last epoch's total sum squared error"`
	EpcAvgSSE  float64 `inactive:"+" desc:"last epoch's average sum squared error (average over trials, and over units within layer)"`
	EpcPctErr  float64 `inactive:"+" desc:"last epoch's percent of trials that had SSE > 0 (subject to .5 unit-wise tolerance)"`
	EpcPctCor  float64 `inactive:"+" desc:"last epoch's percent of trials that had SSE == 0 (subject to .5 unit-wise tolerance)"`
	EpcCosDiff float64 `inactive:"+" desc:"last epoch's average cosine difference for output layer (a normalized error measure, maximum of 1 when the minus phase exactly matches the plus)"`
	FirstZero  int     `inactive:"+" desc:"epoch at when SSE first went to zero"`

	// internal state - view:"-"
	SumSSE       float64          `view:"-" inactive:"+" desc:"sum to increment as we go through epoch"`
	SumAvgSSE    float64          `view:"-" inactive:"+" desc:"sum to increment as we go through epoch"`
	SumCosDiff   float64          `view:"-" inactive:"+" desc:"sum to increment as we go through epoch"`
	CntErr       int              `view:"-" inactive:"+" desc:"sum of errs to increment as we go through epoch"`
	Win          *gi.Window       `view:"-" desc:"main GUI window"`
	NetView      *netview.NetView `view:"-" desc:"the network viewer"`
	ToolBar      *gi.ToolBar      `view:"-" desc:"the master toolbar"`
	TrnTrlPlot   *eplot.Plot2D    `view:"-" desc:"the training trial plot"`
	TrnEpcPlot   *eplot.Plot2D    `view:"-" desc:"the training epoch plot"`
	TstEpcPlot   *eplot.Plot2D    `view:"-" desc:"the testing epoch plot"`
	TstTrlPlot   *eplot.Plot2D    `view:"-" desc:"the test-trial plot"`
	TstCycPlot   *eplot.Plot2D    `view:"-" desc:"the test-cycle plot"`
	RunPlot      *eplot.Plot2D    `view:"-" desc:"the run plot"`
	TrnEpcFile   *os.File         `view:"-" desc:"log file"`
	RunFile      *os.File         `view:"-" desc:"log file"`
	TmpVals      []float32        `view:"-" desc:"temp slice for holding values -- prevent mem allocs"`
	SaveWts      bool             `view:"-" desc:"for command-line run only, auto-save final weights after each run"`
	NoGui        bool             `view:"-" desc:"if true, runing in no GUI mode"`
	LogSetParams bool             `view:"-" desc:"if true, print message for all params that are set"`
	IsRunning    bool             `view:"-" desc:"true if sim is running"`
	StopNow      bool             `view:"-" desc:"flag to stop running"`
	RndSeed      int64            `view:"-" desc:"the current random seed"`

	// DS: vars for storing seed tag
	DirSeed int64 `view:"-" desc:"the seed tag for output data directory"`
}

// KiT_Sim registers this Sim Type and gives it properties that e.g.,
// prompt for filename for save methods.
var KiT_Sim = kit.Types.AddType(&Sim{}, SimProps)

// TheSim is the overall state for this simulation
var TheSim Sim

// New creates new blank elements and initializes defaults
func (ss *Sim) New() {
	ss.Net = &leabra.Network{}
	ss.TrainSL = &etable.Table{}
	ss.TrainEpisodic = &etable.Table{}
	ss.TestSL = &etable.Table{}
	ss.TrnTrlLog = &etable.Table{}
	ss.TrnEpcLog = &etable.Table{}
	ss.TstEpcLog = &etable.Table{}
	ss.TstTrlLog = &etable.Table{}
	ss.TstCycLog = &etable.Table{}
	ss.RunLog = &etable.Table{}
	ss.RunStats = &etable.Table{}
	// DS: Using the params.go paramset
	//	ss.Params = ParamSets
	ss.Params = SavedParamsSets
	ss.RndSeed = 0
	ss.ViewOn = true
	ss.TrainUpdt = leabra.AlphaCycle
	ss.TestUpdt = leabra.AlphaCycle
	ss.TestInterval = 1
	ss.LogSetParams = false
	ss.MemThr = 0.34
	ss.LayStatNms = []string{"ECin", "DG", "CA3", "CA1"}
	ss.TrialperEpc = 80
}

////////////////////////////////////////////////////////////////////////////////////////////
// 		Configs

// Config configures all the elements using the standard functions
func (ss *Sim) Config() {
	ss.OpenPats()
	ss.ConfigEnv()
	ss.ConfigNet(ss.Net)
	ss.ConfigTrnTrlLog(ss.TrnTrlLog)
	ss.ConfigTrnEpcLog(ss.TrnEpcLog)
	ss.ConfigTstEpcLog(ss.TstEpcLog)
	ss.ConfigTstTrlLog(ss.TstTrlLog)
	ss.ConfigTstCycLog(ss.TstCycLog)
	ss.ConfigRunLog(ss.RunLog)
}

// ConfigEnv configures all environments and allows specification for environment variables
func (ss *Sim) ConfigEnv() {
	if ss.MaxRuns == 0 { // allow user override
		ss.MaxRuns = 10
	}
	if ss.MaxEpcs == 0 { // allow user override
		ss.MaxEpcs = 50
	}

	ss.TrainEnv.Nm = "TrainEnv"
	ss.TrainEnv.Dsc = "training params and state"
	ss.TrainEnv.Table = etable.NewIdxView(ss.TrainSL)
	ss.TrainEnv.Validate()
	ss.TrainEnv.Run.Max = ss.MaxRuns // note: we are not setting epoch max -- do that manually
	ss.TrainEnv.Trial.Max = ss.TrialperEpc

	ss.TestEnv.Nm = "TestEnv"
	ss.TestEnv.Dsc = "testing params and state"
	ss.TestEnv.Table = etable.NewIdxView(ss.TestSL)
	ss.TestEnv.Sequential = true
	ss.TestEnv.Validate()

	ss.TrainEnv.Init(0)
	ss.TestEnv.Init(0)
}

// SetEnv select which set of patterns to train on: SL or Episodic
func (ss *Sim) SetEnv(trainAC bool) {
	if trainAC {
		ss.TrainEnv.Table = etable.NewIdxView(ss.TrainEpisodic)
	} else {
		ss.TrainEnv.Table = etable.NewIdxView(ss.TrainSL)
	}
	ss.TrainEnv.Init(0)
}

// ConfigNet configures the leabra network with all layer and projection parameters
func (ss *Sim) ConfigNet(net *leabra.Network) {
	ss.NewRndSeed() // DS added
	net.InitName(net, "Hip")
	in := net.AddLayer2D("Input", 8, 1, emer.Input)
	ecin := net.AddLayer2D("ECin", 8, 1, emer.Hidden)
	ecout := net.AddLayer2D("ECout", 8, 1, emer.Target)
	ca1 := net.AddLayer2D("CA1", 10, 10, emer.Hidden)
	dg := net.AddLayer2D("DG", 20, 20, emer.Hidden)
	ca3 := net.AddLayer2D("CA3", 8, 10, emer.Hidden)

	ecin.SetClass("EC")
	ecout.SetClass("EC")

	ecin.SetRelPos(relpos.Rel{Rel: relpos.RightOf, Other: "Input", YAlign: relpos.Front, Space: 2})
	ecout.SetRelPos(relpos.Rel{Rel: relpos.RightOf, Other: "ECin", YAlign: relpos.Front, Space: 2})
	dg.SetRelPos(relpos.Rel{Rel: relpos.Above, Other: "Input", YAlign: relpos.Front, XAlign: relpos.Left, Space: 0})
	ca3.SetRelPos(relpos.Rel{Rel: relpos.Above, Other: "DG", YAlign: relpos.Front, XAlign: relpos.Left, Space: 0})
	ca1.SetRelPos(relpos.Rel{Rel: relpos.RightOf, Other: "CA3", YAlign: relpos.Front, Space: 2})

	pj := net.ConnectLayersPrjn(in, ecin, prjn.NewOneToOne(), emer.Forward, &hip.EcCa1Prjn{})
	pj.SetClass("EcCa1Prjn")
	pj = net.ConnectLayersPrjn(ecout, ecin, prjn.NewOneToOne(), emer.Back, &hip.EcCa1Prjn{})
	pj.SetClass("EcCa1Prjn")

	// EC <-> CA1 encoder pathways
	enpath := prjn.NewFull()
	pj = net.ConnectLayersPrjn(ecin, ca1, enpath, emer.Forward, &hip.EcCa1Prjn{})
	pj.SetClass("EcCa1Prjn")
	pj = net.ConnectLayersPrjn(ca1, ecout, enpath, emer.Forward, &hip.EcCa1Prjn{})
	pj.SetClass("EcCa1Prjn")
	pj = net.ConnectLayersPrjn(ecout, ca1, enpath, emer.Back, &hip.EcCa1Prjn{})
	pj.SetClass("EcCa1Prjn")

	// Perforant pathway
	ppath := prjn.NewUnifRnd()
	ppath.PCon = 0.25
	ppath.RndSeed = ss.RndSeed //DS added

	pj = net.ConnectLayersPrjn(ecin, dg, ppath, emer.Forward, &hip.CHLPrjn{})
	pj.SetClass("HippoCHL")
	pj = net.ConnectLayersPrjn(ecin, ca3, ppath, emer.Forward, &hip.CHLPrjn{})
	pj.SetClass("HippoCHL")

	// Mossy fibers
	mossy := prjn.NewUnifRnd()
	mossy.PCon = 0.05
	mossy.RndSeed = ss.RndSeed
	pj = net.ConnectLayersPrjn(dg, ca3, mossy, emer.Forward, &hip.CHLPrjn{}) // no learning
	pj.SetClass("HippoCHL")

	// Schafer collaterals
	pj = net.ConnectLayersPrjn(ca3, ca3, prjn.NewFull(), emer.Lateral, &hip.CHLPrjn{})
	pj.SetClass("HippoCHL")
	pj = net.ConnectLayersPrjn(ca3, ca1, prjn.NewFull(), emer.Forward, &hip.CHLPrjn{})
	pj.SetClass("HippoCHL")

	// using 3 threads :)
	dg.SetThread(1)
	ca3.SetThread(2)
	ca1.SetThread(3)

	// note: if you wanted to change a layer type from e.g., Target to Compare, do this:
	// outLay.SetType(emer.Compare)
	// that would mean that the output layer doesn't reflect target values in plus phase
	// and thus removes error-driven learning -- but stats are still computed.

	net.Defaults()
	ss.SetParams("Network", ss.LogSetParams) // only set Network params
	err := net.Build()
	if err != nil {
		log.Println(err)
		return
	}
	net.InitWts()
}

////////////////////////////////////////////////////////////////////////////////
// 	    Init, utils

// Init restarts the run, and initializes everything, including network weights
// and resets the epoch log table
func (ss *Sim) Init() {
	rand.Seed(ss.RndSeed)
	ss.ConfigEnv() // re-config env just in case a different set of patterns was
	// selected or patterns have been modified etc
	ss.StopNow = false
	ss.SetParams("", ss.LogSetParams) // all sheets
	ss.NewRun()
	ss.UpdateView(true)
}

// NewRndSeed gets a new random seed based on current time -- otherwise uses
// the same random seed for every run
func (ss *Sim) NewRndSeed() {
	ss.RndSeed = time.Now().UnixNano()
}

// Counters returns a string of the current counter state
// use tabs to achieve a reasonable formatting overall
// and add a few tabs at the end to allow for expansion..
func (ss *Sim) Counters(train bool) string {
	if train {
		return fmt.Sprintf("Run:\t%d\tEpoch:\t%d\tTrial:\t%d\tCycle:\t%d\tName:\t%v\t\t\t", ss.TrainEnv.Run.Cur, ss.TrainEnv.Epoch.Cur, ss.TrainEnv.Trial.Cur, ss.Time.Cycle, ss.TrainEnv.TrialName)
	}
	return fmt.Sprintf("Run:\t%d\tEpoch:\t%d\tTrial:\t%d\tCycle:\t%d\tName:\t%v\t\t\t", ss.TrainEnv.Run.Cur, ss.TrainEnv.Epoch.Cur, ss.TestEnv.Trial.Cur, ss.Time.Cycle, ss.TestEnv.TrialName)
}

// UpdateView updates the view (DS: What does it actually do?)
func (ss *Sim) UpdateView(train bool) {
	if ss.NetView != nil && ss.NetView.IsVisible() {
		ss.NetView.Record(ss.Counters(train))
		// note: essential to use Go version of update when called from another goroutine
		ss.NetView.GoUpdate() // note: using counters is significantly slower..
	}
}

////////////////////////////////////////////////////////////////////////////////
// 	    Running the Network, starting bottom-up..

// AlphaCyc runs one alpha-cycle (100 msec, 4 quarters)	of processing.
// External inputs must have already been applied prior to calling,
// using ApplyExt method on relevant layers (see TrainTrial, TestTrial).
// If train is true, then learning DWt or WtFmDWt calls are made.
// Handles netview updating within scope of AlphaCycle
func (ss *Sim) AlphaCyc(train bool) {
	// ss.Win.PollEvents() // this can be used instead of running in a separate goroutine
	//fmt.Println(ss.Time)
	viewUpdt := ss.TrainUpdt
	if !train {
		viewUpdt = ss.TestUpdt
	}
	// update prior weight changes at start, so any DWt values remain visible at end
	// you might want to do this less frequently to achieve a mini-batch update
	// in which case, move it out to the TrainTrial method where the relevant
	// counters are being dealt with.
	if train {
		ss.Net.WtFmDWt()
	}

	ca1 := ss.Net.LayerByName("CA1").(*leabra.Layer)
	ca3 := ss.Net.LayerByName("CA3").(*leabra.Layer)
	dg := ss.Net.LayerByName("DG").(*leabra.Layer)
	ecin := ss.Net.LayerByName("ECin").(*leabra.Layer)
	ecout := ss.Net.LayerByName("ECout").(*leabra.Layer)
	ca1FmECin := ca1.RcvPrjns.SendName("ECin").(*hip.EcCa1Prjn)
	ca1FmCa3 := ca1.RcvPrjns.SendName("CA3").(*hip.CHLPrjn)

	// DS: declaring vars to store cycle data for the whole trial
	var ecinTrlCycActs [][]float32
	var ecoutTrlCycActs [][]float32
	var dgTrlCycActs [][]float32
	var ca1TrlCycActs [][]float32
	var ca3TrlCycActs [][]float32

	if train {
		ecout.SetType(emer.Target) // clamp a plus phase during testing
	} else {
		ecout.SetType(emer.Compare) // don't clamp
	}
	ecout.UpdateExtFlags() // call this after updating type DS: added from a later version of hip.go

	// First Quarter: CA1 is driven by ECin, not by CA3 recall
	// (which is not really active yet anyway)
	if train {
		ca1FmECin.WtScale.Abs = 1
		ca1FmCa3.WtScale.Abs = 0
	}

	ss.Net.AlphaCycInit()
	ss.Time.AlphaCycStart()
	for qtr := 0; qtr < 4; qtr++ {
		for cyc := 0; cyc < 25; cyc++ {
			ss.Net.Cycle(&ss.Time)
			if !train {
				ss.LogTstCyc(ss.TstCycLog, ss.Time.Cycle)
			}
			ss.Time.CycleInc()
			if ss.ViewOn {
				switch viewUpdt {
				case leabra.Cycle:
					ss.UpdateView(train)
				case leabra.FastSpike:
					if (cyc+1)%10 == 0 {
						ss.UpdateView(train)
					}
				}
			}
			// Getting cycle activation data here

			var ecinTrlCycAct []float32
			var ecoutTrlCycAct []float32
			var dgTrlCycAct []float32
			var ca1TrlCycAct []float32
			var ca3TrlCycAct []float32
			ecin.UnitVals(&ecinTrlCycAct, "Act")
			ecinTrlCycActs = append(ecinTrlCycActs, ecinTrlCycAct)
			ecout.UnitVals(&ecoutTrlCycAct, "Act")
			ecoutTrlCycActs = append(ecoutTrlCycActs, ecoutTrlCycAct)
			dg.UnitVals(&dgTrlCycAct, "Act")
			dgTrlCycActs = append(dgTrlCycActs, dgTrlCycAct)
			ca1.UnitVals(&ca1TrlCycAct, "Act")
			ca1TrlCycActs = append(ca1TrlCycActs, ca1TrlCycAct)
			ca3.UnitVals(&ca3TrlCycAct, "Act")
			ca3TrlCycActs = append(ca3TrlCycActs, ca3TrlCycAct)
		}

		switch qtr + 1 {
		case 1: // Second, Third Quarters: CA1 is driven by CA3 recall
			if train {
				ca1FmECin.WtScale.Abs = 0
				ca1FmCa3.WtScale.Abs = 1
				ss.Net.GScaleFmAvgAct() // update computed scaling factors
				ss.Net.InitGInc()       // scaling params change, so need to recompute all netins
			}
		case 3: // Fourth Quarter: CA1 back to ECin drive only
			if train {
				ca1FmECin.WtScale.Abs = 1
				ca1FmCa3.WtScale.Abs = 0
				ss.Net.GScaleFmAvgAct() // update computed scaling factors
				ss.Net.InitGInc()       // scaling params change, so need to recompute all netins
			}
			if train { // clamp ECout from ECin
				ecin.UnitVals(&ss.TmpVals, "Act")
				ecout.ApplyExt1D32(ss.TmpVals)
			}
		}
		ss.Net.QuarterFinal(&ss.Time)
		if qtr+1 == 3 {
			ss.MemStats(train) // must come after QuarterFinal
		}
		ss.Time.QuarterInc()
		if ss.ViewOn {
			switch {
			case viewUpdt <= leabra.Quarter:
				ss.UpdateView(train)
			case viewUpdt == leabra.Phase:
				if qtr >= 2 {
					ss.UpdateView(train)
				}
			}
		}

	}
	if train {
		ss.Net.DWt()
	}
	if ss.ViewOn && viewUpdt == leabra.AlphaCycle {
		ss.UpdateView(train)
	}
	if !train {
		ss.TstCycPlot.GoUpdate() // make sure up-to-date at end
	}

	// DS: Writing to files here ---------------------------------------------------------------------------------------------------------------------------------
	wts := false     // DS: Training Wts.
	dwts := false    // DS: Training DWts.
	tstacts := true  // DS: Testing acts.
	trnacts := false // DS: Training acts. Massive outputs if writing out all cycles. Be careful.

	// Setting Directory Seed for unique dir name
	if ss.TrainEnv.Run.Cur == 0 {
		ss.DirSeed = ss.RndSeed
	}
	if wts {
		if train {
			var ecintoca1wts []float32
			//var ca3toca1wts []float32
			ca1FmECin.SynVals(&ecintoca1wts, "Wt")
			//ca1FmCa3.SynVals(&ca3toca1wts, "Wt")

			dirpathwts := ("output\\" + "outputwts" + "\\" + fmt.Sprint(ss.DirSeed) + "_runs_" + fmt.Sprint(ss.MaxRuns))

			if _, err := os.Stat(dirpathwts); os.IsNotExist(err) {
				os.MkdirAll(dirpathwts, os.ModePerm)
			}

			filewts, _ := os.OpenFile(dirpathwts+"\\"+"wts"+fmt.Sprint(ss.RndSeed)+"_"+"run"+fmt.Sprint(ss.TrainEnv.Run.Cur)+"epoch"+fmt.Sprint(ss.TrainEnv.Epoch.Cur)+".csv", os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
			defer filewts.Close()
			writerwts := csv.NewWriter(filewts)
			defer writerwts.Flush()

			if ss.TrainEnv.Trial.Cur == 0 {
				headers := []string{"Run", "Epoch", "Trial", "TrialName"}

				for i := 0; i < 8; i++ {
					for j := 0; j < 100; j++ {
						str := "EC_in" + fmt.Sprint(i) + "to" + "CA1" + fmt.Sprint(j)
						headers = append(headers, str)
					}
				}
				writerwts.Write(headers)
			}

			trlwts := []string{fmt.Sprint(ss.TrainEnv.Run.Cur), fmt.Sprint(ss.TrainEnv.Epoch.Cur), fmt.Sprint(ss.TrainEnv.Trial.Cur), fmt.Sprint(ss.TrainEnv.TrialName)}

			for _, val := range ecintoca1wts {
				trlwts = append(trlwts, fmt.Sprint(val))
			}
			writerwts.Write(trlwts)
		}
	}
	if dwts {
		if train {
			// Getting DWts for CA1 Inverse Checkerboard Investigation
			var ecintoca1dwts []float32
			//var ca3toca1dwts []float32
			ca1FmECin.SynVals(&ecintoca1dwts, "DWt")
			//ca1FmCa3.SynVals(&ca3toca1dwts, "DWt")

			dirpathdwts := ("output\\" + "outputdwts" + "\\" + fmt.Sprint(ss.DirSeed) + "_runs_" + fmt.Sprint(ss.MaxRuns))

			if _, err := os.Stat(dirpathdwts); os.IsNotExist(err) {
				os.MkdirAll(dirpathdwts, os.ModePerm)
			}

			filedwts, _ := os.OpenFile(dirpathdwts+"\\"+"dwts"+fmt.Sprint(ss.RndSeed)+"_"+"run"+fmt.Sprint(ss.TrainEnv.Run.Cur)+"epoch"+fmt.Sprint(ss.TrainEnv.Epoch.Cur)+".csv", os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
			defer filedwts.Close()
			writerdwts := csv.NewWriter(filedwts)
			defer writerdwts.Flush()
			if ss.TrainEnv.Trial.Cur == 0 {
				headers := []string{"Run", "Epoch", "Trial", "TrialName"}

				for i := 0; i < 8; i++ {
					for j := 0; j < 100; j++ {
						str := "EC_in" + fmt.Sprint(i) + "to" + "CA1" + fmt.Sprint(j)
						headers = append(headers, str)
					}
				}
				writerdwts.Write(headers)
			}
			trldwts := []string{fmt.Sprint(ss.TrainEnv.Run.Cur), fmt.Sprint(ss.TrainEnv.Epoch.Cur), fmt.Sprint(ss.TrainEnv.Trial.Cur), fmt.Sprint(ss.TrainEnv.TrialName)}

			for _, val := range ecintoca1dwts {
				trldwts = append(trldwts, fmt.Sprint(val))
			}
			writerdwts.Write(trldwts)
		}
	}
	if trnacts {
		if train {
			dirpathtrnacts := ("output\\" + "outputtrnacts" + "\\" + fmt.Sprint(ss.DirSeed) + "_runs_" + fmt.Sprint(ss.MaxRuns))

			if _, err := os.Stat(dirpathtrnacts); os.IsNotExist(err) {
				os.MkdirAll(dirpathtrnacts, os.ModePerm)
			}

			filetrnacts, _ := os.OpenFile(dirpathtrnacts+"\\"+"trnacts"+fmt.Sprint(ss.RndSeed)+"_"+"run"+fmt.Sprint(ss.TrainEnv.Run.Cur)+"epoch"+fmt.Sprint(ss.TrainEnv.Epoch.Cur)+".csv", os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
			defer filetrnacts.Close()
			writertrnacts := csv.NewWriter(filetrnacts)
			defer writertrnacts.Flush()

			if ss.TrainEnv.Trial.Cur == 0 {
				headers := []string{"Run", "Epoch", "Cycle", "TrialName", "Ecin_0", "Ecin_1",
					"Ecin_2", "Ecin_3", "Ecin_4", "Ecin_5", "Ecin_6", "Ecin_7", "Ecout_0", "Ecout_1",
					"Ecout_2", "Ecout_3", "Ecout_4", "Ecout_5", "Ecout_6", "Ecout_7"}

				for i := 0; i < 400; i++ {
					str := "DG_" + fmt.Sprint(i)
					headers = append(headers, str)
				}
				for i := 0; i < 80; i++ {
					str := "CA3_" + fmt.Sprint(i)
					headers = append(headers, str)
				}
				for i := 0; i < 100; i++ {
					str := "CA1_" + fmt.Sprint(i)
					headers = append(headers, str)
				}
				writertrnacts.Write(headers)
			}

			for i := 0; i < 100; i++ {
				//if i == 20 || i == 95 {
				valueStr := []string{fmt.Sprint(ss.TrainEnv.Run.Cur), fmt.Sprint(ss.TrainEnv.Epoch.Cur), fmt.Sprint(i), fmt.Sprint(ss.TrainEnv.TrialName)}
				for _, vals := range ecinTrlCycActs[i] {
					valueStr = append(valueStr, fmt.Sprint(vals))
				}
				for _, vals := range ecoutTrlCycActs[i] {
					valueStr = append(valueStr, fmt.Sprint(vals))
				}
				for _, vals := range dgTrlCycActs[i] {
					valueStr = append(valueStr, fmt.Sprint(vals))
				}
				for _, vals := range ca3TrlCycActs[i] {
					valueStr = append(valueStr, fmt.Sprint(vals))
				}
				for _, vals := range ca1TrlCycActs[i] {
					valueStr = append(valueStr, fmt.Sprint(vals))
				}
				writertrnacts.Write(valueStr)
				//}
			}

		}
	}
	if tstacts {
		if (!train) && (ss.TestEnv.Trial.Cur < 8) {

			dirpathacts := ("output\\" + "outputacts" + "\\" + "tstacts" + fmt.Sprint(ss.DirSeed) + "_runs_" + fmt.Sprint(ss.MaxRuns))

			if _, err := os.Stat(dirpathacts); os.IsNotExist(err) {
				os.MkdirAll(dirpathacts, os.ModePerm)
			}
			// copying params.go to better track params associated with the run data
			paramsdata, err := ioutil.ReadFile("params.go")
			if err != nil {
				fmt.Println(err)
				return
			}

			err = ioutil.WriteFile(dirpathacts+"\\"+"tstacts"+fmt.Sprint(ss.DirSeed)+"_"+"runs_"+fmt.Sprint(ss.MaxRuns)+"params.go", paramsdata, 0644)
			if err != nil {
				fmt.Println("Error creating", dirpathacts+"\\"+fmt.Sprint(ss.DirSeed)+"_"+"params.go")
				fmt.Println(err)
				return
			}

			filew, _ := os.OpenFile(dirpathacts+"\\"+"tstacts"+fmt.Sprint(ss.RndSeed)+"_"+"run"+fmt.Sprint(ss.TrainEnv.Run.Cur)+"epoch"+fmt.Sprint(ss.TrainEnv.Epoch.Cur)+".csv", os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
			defer filew.Close()
			writerw := csv.NewWriter(filew)
			defer writerw.Flush()

			if ss.TestEnv.TrialName == "A" {
				headers := []string{"Run", "Epoch", "Cycle", "TrialName", "Ecin_0", "Ecin_1",
					"Ecin_2", "Ecin_3", "Ecin_4", "Ecin_5", "Ecin_6", "Ecin_7", "Ecout_0", "Ecout_1",
					"Ecout_2", "Ecout_3", "Ecout_4", "Ecout_5", "Ecout_6", "Ecout_7"}

				for i := 0; i < 400; i++ {
					str := "DG_" + fmt.Sprint(i)
					headers = append(headers, str)
				}
				for i := 0; i < 80; i++ {
					str := "CA3_" + fmt.Sprint(i)
					headers = append(headers, str)
				}
				for i := 0; i < 100; i++ {
					str := "CA1_" + fmt.Sprint(i)
					headers = append(headers, str)
				}
				writerw.Write(headers)
			}

			for i := 0; i < 100; i++ {
				if ss.TrainEnv.Epoch.Cur != ss.MaxEpcs {
					if i == 24 || i == 80 {
						valueStr := []string{fmt.Sprint(ss.TrainEnv.Run.Cur), fmt.Sprint(ss.TrainEnv.Epoch.Cur), fmt.Sprint(i), fmt.Sprint(ss.TestEnv.TrialName)}
						for _, vals := range ecinTrlCycActs[i] {
							valueStr = append(valueStr, fmt.Sprint(vals))
						}
						for _, vals := range ecoutTrlCycActs[i] {
							valueStr = append(valueStr, fmt.Sprint(vals))
						}
						for _, vals := range dgTrlCycActs[i] {
							valueStr = append(valueStr, fmt.Sprint(vals))
						}
						for _, vals := range ca3TrlCycActs[i] {
							valueStr = append(valueStr, fmt.Sprint(vals))
						}
						for _, vals := range ca1TrlCycActs[i] {
							valueStr = append(valueStr, fmt.Sprint(vals))
						}
						writerw.Write(valueStr)
					}
				} else {
					valueStr := []string{fmt.Sprint(ss.TrainEnv.Run.Cur), fmt.Sprint(ss.TrainEnv.Epoch.Cur), fmt.Sprint(i), fmt.Sprint(ss.TestEnv.TrialName)}
					for _, vals := range ecinTrlCycActs[i] {
						valueStr = append(valueStr, fmt.Sprint(vals))
					}
					for _, vals := range ecoutTrlCycActs[i] {
						valueStr = append(valueStr, fmt.Sprint(vals))
					}
					for _, vals := range dgTrlCycActs[i] {
						valueStr = append(valueStr, fmt.Sprint(vals))
					}
					for _, vals := range ca3TrlCycActs[i] {
						valueStr = append(valueStr, fmt.Sprint(vals))
					}
					for _, vals := range ca1TrlCycActs[i] {
						valueStr = append(valueStr, fmt.Sprint(vals))
					}
					writerw.Write(valueStr)
				}
			}
		}
	}
}

// ApplyInputs applies input patterns from given environment.
// It is good practice to have this be a separate method with appropriate
// args so that it can be used for various different contexts
// (training, testing, etc).
func (ss *Sim) ApplyInputs(en env.Env) {
	ss.Net.InitExt() // clear any existing inputs -- not strictly necessary if always
	// going to the same layers, but good practice and cheap anyway

	lays := []string{"Input", "ECout"}
	for _, lnm := range lays {
		ly := ss.Net.LayerByName(lnm).(*leabra.Layer)
		pats := en.State(ly.Nm)
		if pats != nil {
			ly.ApplyExt(pats)
		}
	}
}

// TrainTrial runs one trial of training using TrainEnv
func (ss *Sim) TrainTrial() {
	ss.TrainEnv.Step() // the Env encapsulates and manages all counter state

	// DS: Added to have  testing epoch before first epoch of training
	if ss.TrainEnv.Epoch.Cur == 0 && ss.TrainEnv.Trial.Cur == 0 {
		ss.TestAll()
	}
	//DS --

	// Key to query counters FIRST because current state is in NEXT epoch
	// if epoch counter has changed
	epc, _, chg := ss.TrainEnv.Counter(env.Epoch)
	if chg {
		ss.LogTrnEpc(ss.TrnEpcLog)
		if ss.ViewOn && ss.TrainUpdt > leabra.AlphaCycle {
			ss.UpdateView(true)
			//fmt.Println(ss.TrainEnv.Trial.Cur)
		}
		if epc%ss.TestInterval == 0 { // note: epc is *next* so won't trigger first time
			ss.TestAll()
		}
		if epc >= ss.MaxEpcs { // done with training.
			ss.RunEnd()
			if ss.TrainEnv.Run.Incr() { // we are done!
				ss.StopNow = true
				return
			}
			ss.NewRun()
			return
		}
	}

	ss.ApplyInputs(&ss.TrainEnv)
	ss.AlphaCyc(true)   // train
	ss.TrialStats(true) // accumulate
	ss.LogTrnTrl(ss.TrnTrlLog)
}

// RunEnd is called at the end of a run -- save weights, record final log, etc here
func (ss *Sim) RunEnd() {
	ss.LogRun(ss.RunLog)
	if ss.SaveWts {
		fnm := ss.WeightsFileName()
		fmt.Printf("Saving Weights to: %v\n", fnm)
		ss.Net.SaveWtsJSON(gi.FileName(fnm))
	}
}

// NewRun intializes a new run of the model, using the TrainEnv.Run counter
// for the new run value
func (ss *Sim) NewRun() { //net *leabra.Network) {
	ss.NewRndSeed()
	run := ss.TrainEnv.Run.Cur
	ss.TrainEnv.Init(run)
	ss.TestEnv.Init(run)
	ss.Time.Reset()
	ss.Net.InitWts()
	ss.InitStats()
	ss.TrnTrlLog.SetNumRows(0)
	ss.TrnEpcLog.SetNumRows(0)
	ss.TstEpcLog.SetNumRows(0)

	ca3 := ss.Net.LayerByName("CA3").(*leabra.Layer) //DS added
	dg := ss.Net.LayerByName("DG").(*leabra.Layer)   //DS added

	pjecinca3 := ca3.RcvPrjns.SendName("ECin").(*hip.CHLPrjn)
	pjecinca3.Pattern().(*prjn.UnifRnd).RndSeed = ss.RndSeed
	pjecinca3.Build()

	pjecindg := dg.RcvPrjns.SendName("ECin").(*hip.CHLPrjn)
	pjecindg.Pattern().(*prjn.UnifRnd).RndSeed = ss.RndSeed
	pjecindg.Build()

	pjdgca3 := ca3.RcvPrjns.SendName("DG").(*hip.CHLPrjn)
	pjdgca3.Pattern().(*prjn.UnifRnd).RndSeed = ss.RndSeed
	pjdgca3.Build()

	//pj = ca1.RcvPrjns.SendName("ECin").(*hip.EcCa1Prjn)
	ss.Net.InitWts()

	ss.TrainEnv.Trial.Max = ss.TrialperEpc // DS added
}

// InitStats initializes all the statistics, especially important for the
// cumulative epoch stats -- called at start of new run
func (ss *Sim) InitStats() {
	// accumulators
	ss.SumSSE = 0
	ss.SumAvgSSE = 0
	ss.SumCosDiff = 0
	ss.CntErr = 0
	ss.FirstZero = -1
	// clear rest just to make Sim look initialized
	ss.Mem = 0
	ss.TrgOnWasOffAll = 0
	ss.TrgOnWasOffCmp = 0
	ss.TrgOffWasOn = 0
	ss.TrlSSE = 0
	ss.TrlAvgSSE = 0
	ss.EpcSSE = 0
	ss.EpcAvgSSE = 0
	ss.EpcPctErr = 0
	ss.EpcCosDiff = 0
}

// MemStats computes ActM vs. Target on ECout with binary counts
// must be called at end of 3rd quarter so that Targ values are
// for the entire full pattern as opposed to the plus-phase target
// values clamped from ECin activations
func (ss *Sim) MemStats(train bool) {
	ecout := ss.Net.LayerByName("ECout").(*leabra.Layer)
	ecin := ss.Net.LayerByName("ECin").(*leabra.Layer)
	nn := ecout.Shape().Len()
	trgOnWasOffAll := 0.0 // all units
	trgOnWasOffCmp := 0.0 // only those that required completion, missing in ECin
	trgOffWasOn := 0.0    // should have been off
	cmpN := 0.0           // completion target
	trgOnN := 0.0
	trgOffN := 0.0
	for ni := 0; ni < nn; ni++ {
		actm := ecout.UnitVal1D("ActM", ni)
		trg := ecout.UnitVal1D("Targ", ni) // full pattern target
		inact := ecin.UnitVal1D("ActQ1", ni)
		if trg < 0.5 { // trgOff
			trgOffN++
			if actm > 0.5 {
				trgOffWasOn++
			}
		} else { // trgOn
			trgOnN++
			if inact < 0.5 { // missing in ECin -- completion target
				cmpN++
				if actm < 0.5 {
					trgOnWasOffAll++
					trgOnWasOffCmp++
				}
			} else {
				if actm < 0.5 {
					trgOnWasOffAll++
				}
			}
		}
	}
	trgOnWasOffAll /= trgOnN
	trgOffWasOn /= trgOffN
	if train { // no cmp
		if trgOnWasOffAll < ss.MemThr && trgOffWasOn < ss.MemThr {
			ss.Mem = 1
		} else {
			ss.Mem = 0
		}
	} else { // test
		if cmpN > 0 { // should be
			trgOnWasOffCmp /= cmpN
			if trgOnWasOffCmp < ss.MemThr && trgOffWasOn < ss.MemThr {
				ss.Mem = 1
			} else {
				ss.Mem = 0
			}
		}
	}
	ss.TrgOnWasOffAll = trgOnWasOffAll
	ss.TrgOnWasOffCmp = trgOnWasOffCmp
	ss.TrgOffWasOn = trgOffWasOn
}

// TrialStats computes the trial-level statistics and adds them to the epoch accumulators if
// accum is true.  Note that we're accumulating stats here on the Sim side so the
// core algorithm side remains as simple as possible, and doesn't need to worry about
// different time-scales over which stats could be accumulated etc.
// You can also aggregate directly from log data, as is done for testing stats
func (ss *Sim) TrialStats(accum bool) (sse, avgsse, cosdiff float64) {
	outLay := ss.Net.LayerByName("ECout").(*leabra.Layer)
	ss.TrlCosDiff = float64(outLay.CosDiff.Cos)
	ss.TrlSSE, ss.TrlAvgSSE = outLay.MSE(0.5) // 0.5 = per-unit tolerance -- right side of .5
	if accum {
		ss.SumSSE += ss.TrlSSE
		ss.SumAvgSSE += ss.TrlAvgSSE
		ss.SumCosDiff += ss.TrlCosDiff
		if ss.TrlSSE != 0 {
			ss.CntErr++
		}
	}
	return
}

// TrainEpoch runs training trials for remainder of this epoch
func (ss *Sim) TrainEpoch() {
	ss.StopNow = false
	curEpc := ss.TrainEnv.Epoch.Cur
	curTrial := ss.TrainEnv.Trial.Cur
	for {
		ss.TrainTrial()
		if ss.StopNow || ss.TrainEnv.Epoch.Cur != curEpc || curTrial == ss.TrialperEpc {
			break
		}
	}
	ss.Stopped()
}

// TrainRun runs training trials for remainder of run
func (ss *Sim) TrainRun() {
	ss.StopNow = false
	curRun := ss.TrainEnv.Run.Cur
	for {
		//ss.TestTrial()
		ss.TrainTrial()
		if ss.StopNow || ss.TrainEnv.Run.Cur != curRun {
			break
		}
	}
	ss.Stopped()
}

// Train runs the full training from this point onward
func (ss *Sim) Train() {
	ss.StopNow = false
	for {
		ss.TrainTrial()
		if ss.StopNow {
			break
		}
	}
	ss.Stopped()
}

// Stop tells the sim to stop running
func (ss *Sim) Stop() {
	ss.StopNow = true
}

// Stopped is called when a run method stops running -- updates the IsRunning flag and toolbar
func (ss *Sim) Stopped() {
	ss.IsRunning = false
	if ss.Win != nil {
		vp := ss.Win.WinViewport2D()
		vp.BlockUpdates()
		if ss.ToolBar != nil {
			ss.ToolBar.UpdateActions()
		}
		vp.UnblockUpdates()
		vp.SetNeedsFullRender()
	}
}

// SaveWeights saves the network weights -- when called with giv.CallMethod
// it will auto-prompt for filename
func (ss *Sim) SaveWeights(filename gi.FileName) {
	ss.Net.SaveWtsJSON(filename)
}

////////////////////////////////////////////////////////////////////////////////////////////
// Testing

// TestTrial runs one trial of testing -- always sequentially presented inputs
func (ss *Sim) TestTrial() {
	ss.TestEnv.Step()

	// Query counters FIRST
	_, _, chg := ss.TestEnv.Counter(env.Epoch)
	if chg {
		if ss.ViewOn && ss.TestUpdt > leabra.AlphaCycle {
			ss.UpdateView(false)
		}
		return
	}

	ss.ApplyInputs(&ss.TestEnv)
	ss.AlphaCyc(false)   // !train
	ss.TrialStats(false) // !accumulate
	ss.LogTstTrl(ss.TstTrlLog)
}

// TestItem tests given item which is at given index in test item list
func (ss *Sim) TestItem(idx int) {
	cur := ss.TestEnv.Trial.Cur
	ss.TestEnv.Trial.Cur = idx
	ss.TestEnv.SetTrialName()
	ss.ApplyInputs(&ss.TestEnv)
	ss.AlphaCyc(false)   // !train
	ss.TrialStats(false) // !accumulate
	ss.TestEnv.Trial.Cur = cur
}

// TestAll runs through the full set of testing items
func (ss *Sim) TestAll() {
	ss.TestNm = "AB"
	ss.TestEnv.Table = etable.NewIdxView(ss.TestSL)
	ss.TestEnv.Init(ss.TrainEnv.Run.Cur)
	for {
		ss.TestTrial()
		_, _, chg := ss.TestEnv.Counter(env.Epoch)
		if chg || ss.StopNow {
			break
		}
	}
	// log only at very end
	ss.LogTstEpc(ss.TstEpcLog)
}

// RunTestAll runs through the full set of testing items, has stop running = false at end -- for gui
func (ss *Sim) RunTestAll() {
	ss.StopNow = false
	ss.TestAll()
	ss.Stopped()
}

/////////////////////////////////////////////////////////////////////////
//   Params setting

// ParamsName returns name of current set of parameters
func (ss *Sim) ParamsName() string {
	if ss.ParamSet == "" {
		return "Base"
	}
	return ss.ParamSet
}

// SetParams sets the params for "Base" and then current ParamSet.
// If sheet is empty, then it applies all avail sheets (e.g., Network, Sim)
// otherwise just the named sheet
// if setMsg = true then we output a message for each param that was set.
func (ss *Sim) SetParams(sheet string, setMsg bool) error {
	if sheet == "" {
		// this is important for catching typos and ensuring that all sheets can be used
		ss.Params.ValidateSheets([]string{"Network", "Sim"})
	}
	err := ss.SetParamsSet("Base", sheet, setMsg)
	if ss.ParamSet != "" && ss.ParamSet != "Base" {
		err = ss.SetParamsSet(ss.ParamSet, sheet, setMsg)
	}
	return err
}

// SetParamsSet sets the params for given params.Set name.
// If sheet is empty, then it applies all avail sheets (e.g., Network, Sim)
// otherwise just the named sheet
// if setMsg = true then we output a message for each param that was set.
func (ss *Sim) SetParamsSet(setNm string, sheet string, setMsg bool) error {
	pset, err := ss.Params.SetByNameTry(setNm)
	if err != nil {
		return err
	}
	if sheet == "" || sheet == "Network" {
		netp, ok := pset.Sheets["Network"]
		if ok {
			ss.Net.ApplyParams(netp, setMsg)
		}
	}

	if sheet == "" || sheet == "Sim" {
		simp, ok := pset.Sheets["Sim"]
		if ok {
			simp.Apply(ss, setMsg)
		}
	}
	// note: if you have more complex environments with parameters, definitely add
	// sheets for them, e.g., "TrainEnv", "TestEnv" etc
	return err
}

// OpenPat does a conversion from C++ patterns to Go patterns
func (ss *Sim) OpenPat(dt *etable.Table, fname, desc string) {
	err := dt.OpenCSV(gi.FileName(fname), etable.Tab)
	if err != nil {
		log.Println(err)
		return
	}
	dt.SetMetaData("name", strings.TrimSuffix(fname, ".dat"))
	dt.SetMetaData("desc", desc)
}

// OpenPats needs to be called for a  one-time conversion from C++ patterns to Go patterns
func (ss *Sim) OpenPats() {
	//patgen.ReshapeCppFile(ss.TrainAB, "Train_pairs.dat", "Train_pairs_go.dat")
	//patgen.ReshapeCppFile(ss.TrainAC, "Train_pairs_without_transitions.dat", "Train_pairs_without_transitions_go.dat")
	//patgen.ReshapeCppFile(ss.TestAB, "Test_pairs.dat", "Test_pairs_go.dat")
	// patgen.ReshapeCppFile(ss.TestAC, "Test_AC.dat", "TestAC.dat")
	// patgen.ReshapeCppFile(ss.TestLure, "Lure.dat", "TestLure.dat")
	ss.OpenPat(ss.TrainSL, "Train_pairs_go.dat", "SL Training Patterns")
	ss.OpenPat(ss.TrainEpisodic, "Train_pairs_without_transitions_go.dat", "Episodic Training Patterns")
	ss.OpenPat(ss.TestSL, "Test_pairs_go.dat", "SL Testing Patterns")
	//ss.OpenPat(ss.TestAC, "TestAC.dat", "AC Testing Patterns")
	//ss.OpenPat(ss.TestLure, "TestLure.dat", "Lure Testing Patterns")
}

////////////////////////////////////////////////////////////////////////////////////////////
// 		Logging

// RunName returns a name for this run that combines Tag and Params -- add this to
// any file names that are saved.
func (ss *Sim) RunName() string {
	if ss.Tag != "" {
		return ss.Tag + "_" + ss.ParamsName()
	}
	return ss.ParamsName()
}

// RunEpochName returns a string with the run and epoch numbers with leading zeros, suitable
// for using in weights file names.  Uses 3, 5 digits for each.
func (ss *Sim) RunEpochName(run, epc int) string {
	return fmt.Sprintf("%03d_%05d", run, epc)
}

// WeightsFileName returns default current weights file name
func (ss *Sim) WeightsFileName() string {
	return ss.Net.Nm + "_" + ss.RunName() + "_" + ss.RunEpochName(ss.TrainEnv.Run.Cur, ss.TrainEnv.Epoch.Cur) + ".wts"
}

// LogFileName returns default log file name
func (ss *Sim) LogFileName(lognm string) string {
	return ss.Net.Nm + "_" + ss.RunName() + "_" + lognm + ".csv"
}

//////////////////////////////////////////////
//  TrnTrlLog

// LogTrnTrl adds data from current trial to the TrnTrlLog table.
// log always contains number of testing items
func (ss *Sim) LogTrnTrl(dt *etable.Table) {
	epc := ss.TrainEnv.Epoch.Cur
	trl := ss.TrainEnv.Trial.Cur

	row := dt.Rows
	if trl == 0 { // reset at start
		row = 0
	}
	dt.SetNumRows(row + 1)

	dt.SetCellFloat("Run", row, float64(ss.TrainEnv.Run.Cur))
	dt.SetCellFloat("Epoch", row, float64(epc))
	dt.SetCellFloat("Trial", row, float64(trl))
	dt.SetCellString("TrialName", row, ss.TestEnv.TrialName)
	dt.SetCellFloat("SSE", row, ss.TrlSSE)
	dt.SetCellFloat("AvgSSE", row, ss.TrlAvgSSE)
	dt.SetCellFloat("CosDiff", row, ss.TrlCosDiff)

	dt.SetCellFloat("Mem", row, ss.Mem)
	dt.SetCellFloat("TrgOnWasOff", row, ss.TrgOnWasOffAll)
	dt.SetCellFloat("TrgOffWasOn", row, ss.TrgOffWasOn)

	// note: essential to use Go version of update when called from another goroutine
	ss.TrnTrlPlot.GoUpdate()
}

// ConfigTrnTrlLog configures the training trial log
func (ss *Sim) ConfigTrnTrlLog(dt *etable.Table) {

	dt.SetMetaData("name", "TrnTrlLog")
	dt.SetMetaData("desc", "Record of training per input pattern")
	dt.SetMetaData("read-only", "true")
	dt.SetMetaData("precision", strconv.Itoa(LogPrec))

	nt := ss.TestEnv.Table.Len() // number in view
	sch := etable.Schema{
		{"Run", etensor.INT64, nil, nil},
		{"Epoch", etensor.INT64, nil, nil},
		{"Trial", etensor.INT64, nil, nil},
		{"TrialName", etensor.STRING, nil, nil},
		{"SSE", etensor.FLOAT64, nil, nil},
		{"AvgSSE", etensor.FLOAT64, nil, nil},
		{"CosDiff", etensor.FLOAT64, nil, nil},
		{"Mem", etensor.FLOAT64, nil, nil},
		{"TrgOnWasOff", etensor.FLOAT64, nil, nil},
		{"TrgOffWasOn", etensor.FLOAT64, nil, nil},
	}
	dt.SetFromSchema(sch, nt)
}

// ConfigTrnTrlPlot configures the train trial plot
func (ss *Sim) ConfigTrnTrlPlot(plt *eplot.Plot2D, dt *etable.Table) *eplot.Plot2D {
	plt.Params.Title = "Hippocampus Train Trial Plot"
	plt.Params.XAxisCol = "Trial"
	plt.SetTable(dt)
	// order of params: on, fixMin, min, fixMax, max
	plt.SetColParams("Run", false, true, 0, false, 0)
	plt.SetColParams("Epoch", false, true, 0, false, 0)
	plt.SetColParams("Trial", false, true, 0, false, 0)
	plt.SetColParams("TrialName", false, true, 0, false, 0)
	plt.SetColParams("SSE", false, true, 0, false, 0)
	plt.SetColParams("AvgSSE", false, true, 0, false, 0)
	plt.SetColParams("CosDiff", false, true, 0, true, 1)

	plt.SetColParams("Mem", true, true, 0, true, 1)
	plt.SetColParams("TrgOnWasOff", true, true, 0, true, 1)
	plt.SetColParams("TrgOffWasOn", true, true, 0, true, 1)

	return plt
}

//////////////////////////////////////////////
//  TrnEpcLog

// LogTrnEpc adds data from current epoch to the TrnEpcLog table.
// computes epoch averages prior to logging.
func (ss *Sim) LogTrnEpc(dt *etable.Table) {
	row := dt.Rows
	dt.SetNumRows(row + 1)

	epc := ss.TrainEnv.Epoch.Prv           // this is triggered by increment so use previous value
	nt := float64(ss.TrainEnv.Table.Len()) // number of trials in view

	ss.EpcSSE = ss.SumSSE / nt
	ss.SumSSE = 0
	ss.EpcAvgSSE = ss.SumAvgSSE / nt
	ss.SumAvgSSE = 0
	ss.EpcPctErr = float64(ss.CntErr) / nt
	ss.CntErr = 0
	ss.EpcPctCor = 1 - ss.EpcPctErr
	ss.EpcCosDiff = ss.SumCosDiff / nt
	ss.SumCosDiff = 0
	if ss.FirstZero < 0 && ss.EpcPctErr == 0 {
		ss.FirstZero = epc
	}

	trlog := ss.TrnTrlLog
	tix := etable.NewIdxView(trlog)

	dt.SetCellFloat("Run", row, float64(ss.TrainEnv.Run.Cur))
	dt.SetCellFloat("Epoch", row, float64(epc))
	dt.SetCellFloat("SSE", row, ss.EpcSSE)
	dt.SetCellFloat("AvgSSE", row, ss.EpcAvgSSE)
	dt.SetCellFloat("PctErr", row, ss.EpcPctErr)
	dt.SetCellFloat("PctCor", row, ss.EpcPctCor)
	dt.SetCellFloat("CosDiff", row, ss.EpcCosDiff)

	dt.SetCellFloat("MemPct", row, agg.Mean(tix, "Mem")[0])
	dt.SetCellFloat("TrgOnWasOff", row, agg.Mean(tix, "TrgOnWasOff")[0])
	dt.SetCellFloat("TrgOffWasOn", row, agg.Mean(tix, "TrgOffWasOn")[0])

	for _, lnm := range ss.LayStatNms {
		ly := ss.Net.LayerByName(lnm).(*leabra.Layer)
		dt.SetCellFloat(ly.Nm+" ActAvg", row, float64(ly.Pools[0].ActAvg.ActPAvgEff))
	}

	// note: essential to use Go version of update when called from another goroutine
	ss.TrnEpcPlot.GoUpdate()
	if ss.TrnEpcFile != nil {
		if ss.TrainEnv.Run.Cur == 0 && epc == 0 {
			dt.WriteCSVHeaders(ss.TrnEpcFile, etable.Tab)
		}
		dt.WriteCSVRow(ss.TrnEpcFile, row, etable.Tab, true)
	}
}

// ConfigTrnEpcLog configures the training epoch log
func (ss *Sim) ConfigTrnEpcLog(dt *etable.Table) {
	dt.SetMetaData("name", "TrnEpcLog")
	dt.SetMetaData("desc", "Record of performance over epochs of training")
	dt.SetMetaData("read-only", "true")
	dt.SetMetaData("precision", strconv.Itoa(LogPrec))

	sch := etable.Schema{
		{"Run", etensor.INT64, nil, nil},
		{"Epoch", etensor.INT64, nil, nil},
		{"SSE", etensor.FLOAT64, nil, nil},
		{"AvgSSE", etensor.FLOAT64, nil, nil},
		{"PctErr", etensor.FLOAT64, nil, nil},
		{"PctCor", etensor.FLOAT64, nil, nil},
		{"CosDiff", etensor.FLOAT64, nil, nil},
		{"MemPct", etensor.FLOAT64, nil, nil},
		{"TrgOnWasOff", etensor.FLOAT64, nil, nil},
		{"TrgOffWasOn", etensor.FLOAT64, nil, nil},
	}
	for _, lnm := range ss.LayStatNms {
		sch = append(sch, etable.Column{lnm + " ActAvg", etensor.FLOAT64, nil, nil})
	}
	dt.SetFromSchema(sch, 0)
}

// ConfigTrnEpcPlot configures the training epoch plot
func (ss *Sim) ConfigTrnEpcPlot(plt *eplot.Plot2D, dt *etable.Table) *eplot.Plot2D {
	plt.Params.Title = "Hippocampus Epoch Plot"
	plt.Params.XAxisCol = "Epoch"
	plt.SetTable(dt)
	// order of params: on, fixMin, min, fixMax, max
	plt.SetColParams("Run", false, true, 0, false, 0)
	plt.SetColParams("Epoch", false, true, 0, false, 0)
	plt.SetColParams("SSE", false, true, 0, false, 0)
	plt.SetColParams("AvgSSE", false, true, 0, false, 0)
	plt.SetColParams("PctErr", false, true, 0, true, 1)
	plt.SetColParams("PctCor", false, true, 0, true, 1)
	plt.SetColParams("CosDiff", false, true, 0, true, 1)

	plt.SetColParams("MemPct", true, true, 0, true, 1)      // default plot
	plt.SetColParams("TrgOnWasOff", true, true, 0, true, 1) // default plot
	plt.SetColParams("TrgOffWasOn", true, true, 0, true, 1) // default plot

	for _, lnm := range ss.LayStatNms {
		plt.SetColParams(lnm+" ActAvg", false, true, 0, true, .5)
	}
	return plt
}

//////////////////////////////////////////////
//  TstTrlLog

// LogTstTrl adds data from current trial to the TstTrlLog table.
// log always contains number of testing items
func (ss *Sim) LogTstTrl(dt *etable.Table) {
	epc := ss.TrainEnv.Epoch.Prv // this is triggered by increment so use previous value
	trl := ss.TestEnv.Trial.Cur

	row := dt.Rows
	if ss.TestNm == "AB" && trl == 0 { // reset at start
		row = 0
	}
	dt.SetNumRows(row + 1)

	dt.SetCellFloat("Run", row, float64(ss.TrainEnv.Run.Cur))
	dt.SetCellFloat("Epoch", row, float64(epc))
	dt.SetCellString("TestNm", row, ss.TestNm)
	dt.SetCellFloat("Trial", row, float64(row))
	dt.SetCellString("TrialName", row, ss.TestEnv.TrialName)
	dt.SetCellFloat("SSE", row, ss.TrlSSE)
	dt.SetCellFloat("AvgSSE", row, ss.TrlAvgSSE)
	dt.SetCellFloat("CosDiff", row, ss.TrlCosDiff)

	dt.SetCellFloat("Mem", row, ss.Mem)
	dt.SetCellFloat("TrgOnWasOff", row, ss.TrgOnWasOffCmp)
	dt.SetCellFloat("TrgOffWasOn", row, ss.TrgOffWasOn)

	for _, lnm := range ss.LayStatNms {
		ly := ss.Net.LayerByName(lnm).(*leabra.Layer)
		dt.SetCellFloat(ly.Nm+" ActM.Avg", row, float64(ly.Pools[0].ActM.Avg))
	}

	// note: essential to use Go version of update when called from another goroutine
	ss.TstTrlPlot.GoUpdate()
}

// ConfigTstTrlLog configures the testing trial log
func (ss *Sim) ConfigTstTrlLog(dt *etable.Table) {
	// inLay := ss.Net.LayerByName("Input").(*leabra.Layer)
	// outLay := ss.Net.LayerByName("Output").(*leabra.Layer)

	dt.SetMetaData("name", "TstTrlLog")
	dt.SetMetaData("desc", "Record of testing per input pattern")
	dt.SetMetaData("read-only", "true")
	dt.SetMetaData("precision", strconv.Itoa(LogPrec))

	nt := ss.TestEnv.Table.Len() // number in view
	sch := etable.Schema{
		{"Run", etensor.INT64, nil, nil},
		{"Epoch", etensor.INT64, nil, nil},
		{"TestNm", etensor.STRING, nil, nil},
		{"Trial", etensor.INT64, nil, nil},
		{"TrialName", etensor.STRING, nil, nil},
		{"SSE", etensor.FLOAT64, nil, nil},
		{"AvgSSE", etensor.FLOAT64, nil, nil},
		{"CosDiff", etensor.FLOAT64, nil, nil},
		{"Mem", etensor.FLOAT64, nil, nil},
		{"TrgOnWasOff", etensor.FLOAT64, nil, nil},
		{"TrgOffWasOn", etensor.FLOAT64, nil, nil},
	}
	for _, lnm := range ss.LayStatNms {
		sch = append(sch, etable.Column{lnm + " ActM.Avg", etensor.FLOAT64, nil, nil})
	}
	// sch = append(sch, etable.Schema{
	// 	{"InAct", etensor.FLOAT64, inLay.Shp.Shp, nil},
	// 	{"OutActM", etensor.FLOAT64, outLay.Shp.Shp, nil},
	// 	{"OutActP", etensor.FLOAT64, outLay.Shp.Shp, nil},
	// }...)
	dt.SetFromSchema(sch, nt)
}

// ConfigTstTrlPlot configures the testing trial plot
func (ss *Sim) ConfigTstTrlPlot(plt *eplot.Plot2D, dt *etable.Table) *eplot.Plot2D {
	plt.Params.Title = "Hippocampus Test Trial Plot"
	plt.Params.XAxisCol = "Trial"
	plt.SetTable(dt)
	// order of params: on, fixMin, min, fixMax, max
	plt.SetColParams("Run", false, true, 0, false, 0)
	plt.SetColParams("Epoch", false, true, 0, false, 0)
	plt.SetColParams("TestNm", false, true, 0, false, 0)
	plt.SetColParams("Trial", false, true, 0, false, 0)
	plt.SetColParams("TrialName", false, true, 0, false, 0)
	plt.SetColParams("SSE", false, true, 0, false, 0)
	plt.SetColParams("AvgSSE", false, true, 0, false, 0)
	plt.SetColParams("CosDiff", false, true, 0, true, 1)

	plt.SetColParams("Mem", true, true, 0, true, 1)
	plt.SetColParams("TrgOnWasOff", true, true, 0, true, 1)
	plt.SetColParams("TrgOffWasOn", true, true, 0, true, 1)

	for _, lnm := range ss.LayStatNms {
		plt.SetColParams(lnm+" ActM.Avg", false, true, 0, true, .5)
	}

	// plt.SetColParams("InAct", false, true, 0, true, 1)
	// plt.SetColParams("OutActM", false, true, 0, true, 1)
	// plt.SetColParams("OutActP", false, true, 0, true, 1)
	return plt
}

//////////////////////////////////////////////
//  TstEpcLog

// LogTstEpc adds data from current test epoch to the TstEpcLog table.
func (ss *Sim) LogTstEpc(dt *etable.Table) {
	row := dt.Rows
	dt.SetNumRows(row + 1)

	trl := ss.TstTrlLog
	tix := etable.NewIdxView(trl)
	epc := ss.TrainEnv.Epoch.Prv // ?

	// note: this shows how to use agg methods to compute summary data from another
	// data table, instead of incrementing on the Sim
	dt.SetCellFloat("Run", row, float64(ss.TrainEnv.Run.Cur))
	dt.SetCellFloat("Epoch", row, float64(epc))
	dt.SetCellFloat("SSE", row, agg.Sum(tix, "SSE")[0])
	dt.SetCellFloat("AvgSSE", row, agg.Mean(tix, "AvgSSE")[0])
	dt.SetCellFloat("PctErr", row, agg.PropIf(tix, "SSE", func(idx int, val float64) bool {
		return val > 0
	})[0])
	dt.SetCellFloat("PctCor", row, agg.PropIf(tix, "SSE", func(idx int, val float64) bool {
		return val == 0
	})[0])
	dt.SetCellFloat("CosDiff", row, agg.Mean(tix, "CosDiff")[0])

	trlab := etable.NewIdxView(trl)
	trlab.Filter(func(et *etable.Table, row int) bool {
		return et.CellString("TestNm", row) == "AB"
	})
	//trlac := etable.NewIdxView(trl)
	//trlac.Filter(func(et *etable.Table, row int) bool {
	//	return et.CellString("TestNm", row) == "AC"
	//})
	//trllure := etable.NewIdxView(trl)
	//trllure.Filter(func(et *etable.Table, row int) bool {
	//	return et.CellString("TestNm", row) == "Lure"
	//})

	dt.SetCellFloat("AB MemPct", row, agg.Mean(trlab, "Mem")[0])
	dt.SetCellFloat("AB TrgOnWasOff", row, agg.Mean(trlab, "TrgOnWasOff")[0])
	dt.SetCellFloat("AB TrgOffWasOn", row, agg.Mean(trlab, "TrgOffWasOn")[0])

	//dt.SetCellFloat("AC MemPct", row, agg.Mean(trlac, "Mem")[0])
	//dt.SetCellFloat("AC TrgOnWasOff", row, agg.Mean(trlac, "TrgOnWasOff")[0])
	//dt.SetCellFloat("AC TrgOffWasOn", row, agg.Mean(trlac, "TrgOffWasOn")[0])

	//dt.SetCellFloat("Lure MemPct", row, agg.Mean(trllure, "Mem")[0])
	//dt.SetCellFloat("Lure TrgOnWasOff", row, agg.Mean(trllure, "TrgOnWasOff")[0])
	//dt.SetCellFloat("Lure TrgOffWasOn", row, agg.Mean(trllure, "TrgOffWasOn")[0])

	// note: essential to use Go version of update when called from another goroutine
	ss.TstEpcPlot.GoUpdate()
}

// ConfigTstEpcLog configures the test epoch log
func (ss *Sim) ConfigTstEpcLog(dt *etable.Table) {
	dt.SetMetaData("name", "TstEpcLog")
	dt.SetMetaData("desc", "Summary stats for testing trials")
	dt.SetMetaData("read-only", "true")
	dt.SetMetaData("precision", strconv.Itoa(LogPrec))

	sch := etable.Schema{
		{"Run", etensor.INT64, nil, nil},
		{"Epoch", etensor.INT64, nil, nil},
		{"SSE", etensor.FLOAT64, nil, nil},
		{"AvgSSE", etensor.FLOAT64, nil, nil},
		{"PctErr", etensor.FLOAT64, nil, nil},
		{"PctCor", etensor.FLOAT64, nil, nil},
		{"CosDiff", etensor.FLOAT64, nil, nil},
		{"AB MemPct", etensor.FLOAT64, nil, nil},
		{"AB TrgOnWasOff", etensor.FLOAT64, nil, nil},
		{"AB TrgOffWasOn", etensor.FLOAT64, nil, nil},
		//{"AC MemPct", etensor.FLOAT64, nil, nil},
		//{"AC TrgOnWasOff", etensor.FLOAT64, nil, nil},
		//{"AC TrgOffWasOn", etensor.FLOAT64, nil, nil},
		//{"Lure MemPct", etensor.FLOAT64, nil, nil},
		//{"Lure TrgOnWasOff", etensor.FLOAT64, nil, nil},
		//{"Lure TrgOffWasOn", etensor.FLOAT64, nil, nil},
	}
	dt.SetFromSchema(sch, 0)
}

// ConfigTstEpcPlot configures the test epoch plot
func (ss *Sim) ConfigTstEpcPlot(plt *eplot.Plot2D, dt *etable.Table) *eplot.Plot2D {
	plt.Params.Title = "Hippocampus Testing Epoch Plot"
	plt.Params.XAxisCol = "Epoch"
	plt.SetTable(dt)
	// order of params: on, fixMin, min, fixMax, max
	plt.SetColParams("Run", false, true, 0, false, 0)
	plt.SetColParams("Epoch", false, true, 0, false, 0)
	plt.SetColParams("SSE", false, true, 0, false, 0)
	plt.SetColParams("AvgSSE", false, true, 0, false, 0)
	plt.SetColParams("PctErr", false, true, 0, true, 1)
	plt.SetColParams("PctCor", false, true, 0, true, 1)
	plt.SetColParams("CosDiff", false, true, 0, true, 1)

	plt.SetColParams("AB MemPct", true, true, 0, true, 1) // default plot
	plt.SetColParams("AB TrgOnWasOff", false, true, 0, true, 1)
	plt.SetColParams("AB TrgOffWasOn", false, true, 0, true, 1)

	//plt.SetColParams("AC MemPct", true, true, 0, true, 1) // default plot
	//plt.SetColParams("AC TrgOnWasOff", false, true, 0, true, 1)
	//plt.SetColParams("AC TrgOffWasOn", false, true, 0, true, 1)

	//plt.SetColParams("Lure MemPct", true, true, 0, true, 1)
	//plt.SetColParams("Lure TrgOnWasOff", false, true, 0, true, 1)
	//plt.SetColParams("Lure TrgOffWasOn", false, true, 0, true, 1)

	return plt
}

//////////////////////////////////////////////
//  TstCycLog

// LogTstCyc adds data from current trial to the TstCycLog table.
// log just has 100 cycles, is overwritten
func (ss *Sim) LogTstCyc(dt *etable.Table, cyc int) {
	if dt.Rows <= cyc {
		dt.SetNumRows(cyc + 1)
	}

	dt.SetCellFloat("Cycle", cyc, float64(cyc))
	//for _, lnm := range ss.LayStatNms {
	//ly := ss.Net.LayerByName(lnm).(*leabra.Layer)
	//dt.SetCellFloat(ly.Nm+" Ge.Avg", cyc, float64(ly.Pools[0].Ge.Avg)) //This and the next line broke because of latest change to emer! (16/9/19)
	//dt.SetCellFloat(ly.Nm+" Act.Avg", cyc, float64(ly.Pools[0].Act.Avg))
	//}

	//if cyc%10 == 0 { // too slow to do every cyc
	// note: essential to use Go version of update when called from another goroutine
	//ss.TstCycPlot.GoUpdate()
}

//}

// ConfigTstCycLog configures the test cycle log
func (ss *Sim) ConfigTstCycLog(dt *etable.Table) {
	dt.SetMetaData("name", "TstCycLog")
	dt.SetMetaData("desc", "Record of activity etc over one trial by cycle")
	dt.SetMetaData("read-only", "true")
	dt.SetMetaData("precision", strconv.Itoa(LogPrec))

	np := 100 // max cycles
	sch := etable.Schema{
		{"Cycle", etensor.INT64, nil, nil},
	}
	for _, lnm := range ss.LayStatNms {
		sch = append(sch, etable.Column{lnm + " Ge.Avg", etensor.FLOAT64, nil, nil})
		sch = append(sch, etable.Column{lnm + " Act.Avg", etensor.FLOAT64, nil, nil})
	}
	dt.SetFromSchema(sch, np)
}

// ConfigTstCycPlot configures the test cycle plot
func (ss *Sim) ConfigTstCycPlot(plt *eplot.Plot2D, dt *etable.Table) *eplot.Plot2D {
	plt.Params.Title = "Hippocampus Test Cycle Plot"
	plt.Params.XAxisCol = "Cycle"
	plt.SetTable(dt)
	// order of params: on, fixMin, min, fixMax, max
	plt.SetColParams("Cycle", false, true, 0, false, 0)
	for _, lnm := range ss.LayStatNms {
		plt.SetColParams(lnm+" Ge.Avg", true, true, 0, true, .5)
		plt.SetColParams(lnm+" Act.Avg", true, true, 0, true, .5)
	}
	return plt
}

//////////////////////////////////////////////
//  RunLog

// LogRun adds data from current run to the RunLog table.
func (ss *Sim) LogRun(dt *etable.Table) {
	run := ss.TrainEnv.Run.Cur // this is NOT triggered by increment yet -- use Cur
	row := dt.Rows
	dt.SetNumRows(row + 1)

	epclog := ss.TstEpcLog
	// compute mean over last N epochs for run level
	nlast := 1
	epcix := etable.NewIdxView(epclog)
	epcix.Idxs = epcix.Idxs[epcix.Len()-nlast-1:]

	params := ss.RunName() // includes tag

	dt.SetCellFloat("Run", row, float64(run))
	dt.SetCellString("Params", row, params)
	dt.SetCellFloat("FirstZero", row, float64(ss.FirstZero))
	dt.SetCellFloat("SSE", row, agg.Mean(epcix, "SSE")[0])
	dt.SetCellFloat("AvgSSE", row, agg.Mean(epcix, "AvgSSE")[0])
	dt.SetCellFloat("PctErr", row, agg.Mean(epcix, "PctErr")[0])
	dt.SetCellFloat("PctCor", row, agg.Mean(epcix, "PctCor")[0])
	dt.SetCellFloat("CosDiff", row, agg.Mean(epcix, "CosDiff")[0])

	dt.SetCellFloat("AB MemPct", row, agg.Mean(epcix, "AB MemPct")[0])
	dt.SetCellFloat("AB TrgOnWasOff", row, agg.Mean(epcix, "AB TrgOnWasOff")[0])
	dt.SetCellFloat("AB TrgOffWasOn", row, agg.Mean(epcix, "AB TrgOffWasOn")[0])

	runix := etable.NewIdxView(dt)
	spl := split.GroupBy(runix, []string{"Params"})
	split.Desc(spl, "AB MemPct")
	split.Desc(spl, "AB TrgOnWasOff")
	split.Desc(spl, "AB TrgOffWasOn")
	ss.RunStats = spl.AggsToTable(false)

	// note: essential to use Go version of update when called from another goroutine
	ss.RunPlot.GoUpdate()
	if ss.RunFile != nil {
		if row == 0 {
			dt.WriteCSVHeaders(ss.RunFile, etable.Tab)
		}
		dt.WriteCSVRow(ss.RunFile, row, etable.Tab, true)
	}
}

// ConfigRunLog configures the run log
func (ss *Sim) ConfigRunLog(dt *etable.Table) {
	dt.SetMetaData("name", "RunLog")
	dt.SetMetaData("desc", "Record of performance at end of training")
	dt.SetMetaData("read-only", "true")
	dt.SetMetaData("precision", strconv.Itoa(LogPrec))

	sch := etable.Schema{
		{"Run", etensor.INT64, nil, nil},
		{"Params", etensor.STRING, nil, nil},
		{"FirstZero", etensor.FLOAT64, nil, nil},
		{"SSE", etensor.FLOAT64, nil, nil},
		{"AvgSSE", etensor.FLOAT64, nil, nil},
		{"PctErr", etensor.FLOAT64, nil, nil},
		{"PctCor", etensor.FLOAT64, nil, nil},
		{"CosDiff", etensor.FLOAT64, nil, nil},
		{"AB MemPct", etensor.FLOAT64, nil, nil},
		{"AB TrgOnWasOff", etensor.FLOAT64, nil, nil},
		{"AB TrgOffWasOn", etensor.FLOAT64, nil, nil},
	}
	dt.SetFromSchema(sch, 0)
}

// ConfigRunPlot configures the run plot
func (ss *Sim) ConfigRunPlot(plt *eplot.Plot2D, dt *etable.Table) *eplot.Plot2D {
	plt.Params.Title = "Hippocampus Run Plot"
	plt.Params.XAxisCol = "Run"
	plt.SetTable(dt)
	// order of params: on, fixMin, min, fixMax, max
	plt.SetColParams("Run", false, true, 0, false, 0)
	plt.SetColParams("FirstZero", false, true, 0, false, 0)
	plt.SetColParams("SSE", false, true, 0, false, 0)
	plt.SetColParams("AvgSSE", false, true, 0, false, 0)
	plt.SetColParams("PctErr", false, true, 0, true, 1)
	plt.SetColParams("PctCor", false, true, 0, true, 1)
	plt.SetColParams("CosDiff", false, true, 0, true, 1)

	plt.SetColParams("AB MemPct", true, true, 0, true, 1)      // default plot
	plt.SetColParams("AB TrgOnWasOff", true, true, 0, true, 1) // default plot
	plt.SetColParams("AB TrgOffWasOn", true, true, 0, true, 1) // default plot
	return plt
}

////////////////////////////////////////////////////////////////////////////////////////////
// 		Gui

// ConfigGui configures the GoGi gui interface for this simulation,
func (ss *Sim) ConfigGui() *gi.Window {
	width := 1080
	height := 1200

	gi.SetAppName("hip")
	gi.SetAppAbout(`This demonstrates a basic Hippocampus model in Leabra. See <a href="https://github.com/emer/emergent">emergent on GitHub</a>.</p>`)

	win := gi.NewWindow2D("hip", "Hippocampus AB", width, height, true)
	ss.Win = win

	vp := win.WinViewport2D()
	updt := vp.UpdateStart()

	mfr := win.SetMainFrame()

	tbar := gi.AddNewToolBar(mfr, "tbar")
	tbar.SetStretchMaxWidth()
	ss.ToolBar = tbar

	split := gi.AddNewSplitView(mfr, "split")
	split.Dim = gi.X
	split.SetStretchMaxWidth()
	split.SetStretchMaxHeight()

	sv := giv.AddNewStructView(split, "sv")
	sv.SetStruct(ss)

	tv := gi.AddNewTabView(split, "tv")

	nv := tv.AddNewTab(netview.KiT_NetView, "NetView").(*netview.NetView)
	nv.Var = "Act"
	// nv.Params.ColorMap = "Jet" // default is ColdHot
	// which fares pretty well in terms of discussion here:
	// https://matplotlib.org/tutorials/colors/colormaps.html
	nv.SetNet(ss.Net)
	ss.NetView = nv
	nv.ViewDefaults()

	plt := tv.AddNewTab(eplot.KiT_Plot2D, "TrnTrlPlot").(*eplot.Plot2D)
	ss.TrnTrlPlot = ss.ConfigTrnTrlPlot(plt, ss.TrnTrlLog)

	plt = tv.AddNewTab(eplot.KiT_Plot2D, "TrnEpcPlot").(*eplot.Plot2D)
	ss.TrnEpcPlot = ss.ConfigTrnEpcPlot(plt, ss.TrnEpcLog)

	plt = tv.AddNewTab(eplot.KiT_Plot2D, "TstTrlPlot").(*eplot.Plot2D)
	ss.TstTrlPlot = ss.ConfigTstTrlPlot(plt, ss.TstTrlLog)

	plt = tv.AddNewTab(eplot.KiT_Plot2D, "TstEpcPlot").(*eplot.Plot2D)
	ss.TstEpcPlot = ss.ConfigTstEpcPlot(plt, ss.TstEpcLog)

	plt = tv.AddNewTab(eplot.KiT_Plot2D, "TstCycPlot").(*eplot.Plot2D)
	ss.TstCycPlot = ss.ConfigTstCycPlot(plt, ss.TstCycLog)

	plt = tv.AddNewTab(eplot.KiT_Plot2D, "RunPlot").(*eplot.Plot2D)
	ss.RunPlot = ss.ConfigRunPlot(plt, ss.RunLog)

	split.SetSplits(.3, .7)

	tbar.AddAction(gi.ActOpts{Label: "Init", Icon: "update", Tooltip: "Initialize everything including network weights, and start over.  Also applies current params.", UpdateFunc: func(act *gi.Action) {
		act.SetActiveStateUpdt(!ss.IsRunning)
	}}, win.This(), func(recv, send ki.Ki, sig int64, data interface{}) {
		ss.Init()
		vp.SetNeedsFullRender()
	})

	tbar.AddAction(gi.ActOpts{Label: "Train", Icon: "run", Tooltip: "Starts the network training, picking up from wherever it may have left off.  If not stopped, training will complete the specified number of Runs through the full number of Epochs of training, with testing automatically occuring at the specified interval.",
		UpdateFunc: func(act *gi.Action) {
			act.SetActiveStateUpdt(!ss.IsRunning)
		}}, win.This(), func(recv, send ki.Ki, sig int64, data interface{}) {
		if !ss.IsRunning {
			ss.IsRunning = true
			tbar.UpdateActions()
			// ss.Train()
			go ss.Train()
		}
	})

	tbar.AddAction(gi.ActOpts{Label: "Stop", Icon: "stop", Tooltip: "Interrupts running.  Hitting Train again will pick back up where it left off.", UpdateFunc: func(act *gi.Action) {
		act.SetActiveStateUpdt(ss.IsRunning)
	}}, win.This(), func(recv, send ki.Ki, sig int64, data interface{}) {
		ss.Stop()
	})

	tbar.AddAction(gi.ActOpts{Label: "Step Trial", Icon: "step-fwd", Tooltip: "Advances one training trial at a time.", UpdateFunc: func(act *gi.Action) {
		act.SetActiveStateUpdt(!ss.IsRunning)
	}}, win.This(), func(recv, send ki.Ki, sig int64, data interface{}) {
		if !ss.IsRunning {
			ss.IsRunning = true
			ss.TrainTrial()
			ss.IsRunning = false
			vp.SetNeedsFullRender()
		}
	})

	tbar.AddAction(gi.ActOpts{Label: "Step Epoch", Icon: "fast-fwd", Tooltip: "Advances one epoch (complete set of training patterns) at a time.", UpdateFunc: func(act *gi.Action) {
		act.SetActiveStateUpdt(!ss.IsRunning)
	}}, win.This(), func(recv, send ki.Ki, sig int64, data interface{}) {
		if !ss.IsRunning {
			ss.IsRunning = true
			tbar.UpdateActions()
			go ss.TrainEpoch()
		}
	})

	tbar.AddAction(gi.ActOpts{Label: "Step Run", Icon: "fast-fwd", Tooltip: "Advances one full training Run at a time.", UpdateFunc: func(act *gi.Action) {
		act.SetActiveStateUpdt(!ss.IsRunning)
	}}, win.This(), func(recv, send ki.Ki, sig int64, data interface{}) {
		if !ss.IsRunning {
			ss.IsRunning = true
			tbar.UpdateActions()
			go ss.TrainRun()
		}
	})

	tbar.AddSeparator("test")

	tbar.AddAction(gi.ActOpts{Label: "Test Trial", Icon: "step-fwd", Tooltip: "Runs the next testing trial.", UpdateFunc: func(act *gi.Action) {
		act.SetActiveStateUpdt(!ss.IsRunning)
	}}, win.This(), func(recv, send ki.Ki, sig int64, data interface{}) {
		if !ss.IsRunning {
			ss.IsRunning = true
			ss.TestTrial()
			ss.IsRunning = false
			vp.SetNeedsFullRender()
		}
	})

	tbar.AddAction(gi.ActOpts{Label: "Test Item", Icon: "step-fwd", Tooltip: "Prompts for a specific input pattern name to run, and runs it in testing mode.", UpdateFunc: func(act *gi.Action) {
		act.SetActiveStateUpdt(!ss.IsRunning)
	}}, win.This(), func(recv, send ki.Ki, sig int64, data interface{}) {
		gi.StringPromptDialog(vp, "", "Test Item",
			gi.DlgOpts{Title: "Test Item", Prompt: "Enter the Name of a given input pattern to test (case insensitive, contains given string."},
			win.This(), func(recv, send ki.Ki, sig int64, data interface{}) {
				dlg := send.(*gi.Dialog)
				if sig == int64(gi.DialogAccepted) {
					val := gi.StringPromptDialogValue(dlg)
					idxs := ss.TestEnv.Table.RowsByString("Name", val, true, true) // contains, ignoreCase
					if len(idxs) == 0 {
						gi.PromptDialog(nil, gi.DlgOpts{Title: "Name Not Found", Prompt: "No patterns found containing: " + val}, true, false, nil, nil)
					} else {
						if !ss.IsRunning {
							ss.IsRunning = true
							fmt.Printf("testing index: %v\n", idxs[0])
							ss.TestItem(idxs[0])
							ss.IsRunning = false
							vp.SetNeedsFullRender()
						}
					}
				}
			})
	})

	tbar.AddAction(gi.ActOpts{Label: "Test All", Icon: "fast-fwd", Tooltip: "Tests all of the testing trials.", UpdateFunc: func(act *gi.Action) {
		act.SetActiveStateUpdt(!ss.IsRunning)
	}}, win.This(), func(recv, send ki.Ki, sig int64, data interface{}) {
		if !ss.IsRunning {
			ss.IsRunning = true
			tbar.UpdateActions()
			go ss.RunTestAll()
		}
	})

	tbar.AddAction(gi.ActOpts{Label: "Env", Icon: "gear", Tooltip: "select training input patterns: SL or Episodic."}, win.This(),
		func(recv, send ki.Ki, sig int64, data interface{}) {
			giv.CallMethod(ss, "SetEnv", vp)
		})

	tbar.AddSeparator("log")

	tbar.AddAction(gi.ActOpts{Label: "Reset RunLog", Icon: "reset", Tooltip: "Reset the accumulated log of all Runs, which are tagged with the ParamSet used"}, win.This(),
		func(recv, send ki.Ki, sig int64, data interface{}) {
			ss.RunLog.SetNumRows(0)
			ss.RunPlot.Update()
		})

	tbar.AddSeparator("misc")

	tbar.AddAction(gi.ActOpts{Label: "New Seed", Icon: "new", Tooltip: "Generate a new initial random seed to get different results.  By default, Init re-establishes the same initial seed every time."}, win.This(),
		func(recv, send ki.Ki, sig int64, data interface{}) {
			ss.NewRndSeed()
		})

	tbar.AddAction(gi.ActOpts{Label: "README", Icon: "file-markdown", Tooltip: "Opens your browser on the README file that contains instructions for how to run this model."}, win.This(),
		func(recv, send ki.Ki, sig int64, data interface{}) {
			gi.OpenURL("https://github.com/emer/leabra/blob/master/examples/ra25/README.md")
		})

	vp.UpdateEndNoSig(updt)

	// main menu
	appnm := gi.AppName()
	mmen := win.MainMenu
	mmen.ConfigMenus([]string{appnm, "File", "Edit", "Window"})

	amen := win.MainMenu.ChildByName(appnm, 0).(*gi.Action)
	amen.Menu.AddAppMenu(win)

	emen := win.MainMenu.ChildByName("Edit", 1).(*gi.Action)
	emen.Menu.AddCopyCutPaste(win)

	// note: Command in shortcuts is automatically translated into Control for
	// Linux, Windows or Meta for MacOS
	// fmen := win.MainMenu.ChildByName("File", 0).(*gi.Action)
	// fmen.Menu.AddAction(gi.ActOpts{Label: "Open", Shortcut: "Command+O"},
	// 	win.This(), func(recv, send ki.Ki, sig int64, data interface{}) {
	// 		FileViewOpenSVG(vp)
	// 	})
	// fmen.Menu.AddSeparator("csep")
	// fmen.Menu.AddAction(gi.ActOpts{Label: "Close Window", Shortcut: "Command+W"},
	// 	win.This(), func(recv, send ki.Ki, sig int64, data interface{}) {
	// 		win.Close()
	// 	})

	inQuitPrompt := false
	gi.SetQuitReqFunc(func() {
		if inQuitPrompt {
			return
		}
		inQuitPrompt = true
		gi.PromptDialog(vp, gi.DlgOpts{Title: "Really Quit?",
			Prompt: "Are you <i>sure</i> you want to quit and lose any unsaved params, weights, logs, etc?"}, true, true,
			win.This(), func(recv, send ki.Ki, sig int64, data interface{}) {
				if sig == int64(gi.DialogAccepted) {
					gi.Quit()
				} else {
					inQuitPrompt = false
				}
			})
	})

	// gi.SetQuitCleanFunc(func() {
	// 	fmt.Printf("Doing final Quit cleanup here..\n")
	// })

	inClosePrompt := false
	win.SetCloseReqFunc(func(w *gi.Window) {
		if inClosePrompt {
			return
		}
		inClosePrompt = true
		gi.PromptDialog(vp, gi.DlgOpts{Title: "Really Close Window?",
			Prompt: "Are you <i>sure</i> you want to close the window?  This will Quit the App as well, losing all unsaved params, weights, logs, etc"}, true, true,
			win.This(), func(recv, send ki.Ki, sig int64, data interface{}) {
				if sig == int64(gi.DialogAccepted) {
					gi.Quit()
				} else {
					inClosePrompt = false
				}
			})
	})

	win.SetCloseCleanFunc(func(w *gi.Window) {
		go gi.Quit() // once main window is closed, quit
	})

	win.MainMenuUpdated()
	return win
}

// SimProps provides props to register Save methods so they can be used
var SimProps = ki.Props{
	"CallMethods": ki.PropSlice{
		{"SaveWeights", ki.Props{
			"desc": "save network weights to file",
			"icon": "file-save",
			"Args": ki.PropSlice{
				{"File Name", ki.Props{
					"ext": ".wts,.wts.gz",
				}},
			},
		}},
		{"SetEnv", ki.Props{
			"desc": "select which set of patterns to train on: SL or Episodic",
			"icon": "gear",
			"Args": ki.PropSlice{
				{"Train on AC", ki.Props{}},
			},
		}},
	},
}

// CmdArgs checks for cmd args for NoGui functioning
func (ss *Sim) CmdArgs() {
	ss.NoGui = true
	var nogui bool
	var saveEpcLog bool
	var saveRunLog bool
	flag.StringVar(&ss.ParamSet, "params", "", "ParamSet name to use -- must be valid name as listed in compiled-in params or loaded params")
	flag.StringVar(&ss.Tag, "tag", "", "extra tag to add to file names saved from this run")
	flag.IntVar(&ss.MaxRuns, "runs", 300, "number of runs to do (note that MaxEpcs is in paramset)")
	flag.BoolVar(&ss.LogSetParams, "setparams", false, "if true, print a record of each parameter that is set")
	flag.BoolVar(&ss.SaveWts, "wts", false, "if true, save final weights after each run")
	flag.BoolVar(&saveEpcLog, "epclog", true, "if true, save train epoch log to file")
	flag.BoolVar(&saveRunLog, "runlog", true, "if true, save run epoch log to file")
	flag.BoolVar(&nogui, "nogui", true, "if not passing any other args and want to run nogui, use nogui")
	flag.Parse()
	ss.Init()

	if ss.ParamSet != "" {
		fmt.Printf("Using ParamSet: %s\n", ss.ParamSet)
	}

	if saveEpcLog {
		var err error
		fnm := ss.LogFileName("epc")
		ss.TrnEpcFile, err = os.Create(fnm)
		if err != nil {
			log.Println(err)
			ss.TrnEpcFile = nil
		} else {
			fmt.Printf("Saving epoch log to: %v\n", fnm)
			defer ss.TrnEpcFile.Close()
		}
	}
	if saveRunLog {
		var err error
		fnm := ss.LogFileName("run")
		ss.RunFile, err = os.Create(fnm)
		if err != nil {
			log.Println(err)
			ss.RunFile = nil
		} else {
			fmt.Printf("Saving run log to: %v\n", fnm)
			defer ss.RunFile.Close()
		}
	}
	if ss.SaveWts {
		fmt.Printf("Saving final weights per run\n")
	}
	fmt.Printf("Running %d Runs\n", ss.MaxRuns)
	ss.Train()
}

func mainrun() {
	TheSim.New()
	TheSim.Config()

	if len(os.Args) > 1 {
		TheSim.CmdArgs() // simple assumption is that any args = no gui -- could add explicit arg if you want
	} else {
		// gi.Update2DTrace = true
		TheSim.Init()
		win := TheSim.ConfigGui()
		win.StartEventLoop()
	}
}
