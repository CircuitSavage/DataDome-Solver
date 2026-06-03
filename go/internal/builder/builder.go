package builder

import (
	"math/rand"
	"net/url"
	"time"

	ddcrypto "github.com/CircuitSavage/datadome-solver/internal/crypto"
)

// Options configures fingerprint generation.
type Options struct {
	Profile      string
	URL          string
	TagsJSURL    string
	ServerHash   *string
	BPC          int
	Overrides    map[string]any
}

// BuildPayload returns ordered signals for encryption.
func BuildPayload(opts Options) []ddcrypto.Signal {
	siteURL := opts.URL
	tagsURL := opts.TagsJSURL
	if tagsURL == "" && siteURL != "" {
		if parsed, err := url.Parse(siteURL); err == nil && parsed.Host != "" {
			tagsURL = parsed.Scheme + "://" + parsed.Host + "/include/tags.js"
		}
	}

	prof := getProfile(opts.Profile)
	if prof == nil {
		prof = copyProfile(chromeWin10)
	}

	nowMs := time.Now().UnixMilli()
	jset := nowMs / 1000
	trrd := generateTrrd()
	navTiming := generateNavTiming()
	errorStacks := generateErrorStacks(tagsURL)
	wdifpnh := generateWdifpnh()
	r3n := generateR3n(opts.ServerHash)
	bchk := generateBchk()
	dffls := ""

	brW := intVal(prof["br_w"]) + rand.Intn(101) - 50
	brH := intVal(prof["br_h"]) + rand.Intn(101) - 50

	payload := make(map[string]any)

	payload["log2"] = "gl,tzp"
	payload["r3n"] = r3n
	payload["glvd"] = prof["glvd"]
	payload["glrd"] = prof["glrd"]
	payload["nddc"] = 1
	payload["exp8"] = 0
	payload["plu"] = prof["plu"]
	payload["plgod"] = prof["plgod"]
	payload["plg"] = prof["plg"]
	payload["plgne"] = prof["plgne"]
	payload["plgre"] = prof["plgre"]
	payload["plgof"] = prof["plgof"]
	payload["plggt"] = prof["plggt"]
	payload["bfr"] = false
	payload["hdn"] = false
	payload["br_w"] = brW
	payload["br_h"] = brH
	payload["br_iw"] = brW
	payload["br_ih"] = brH
	payload["ars_w"] = prof["ars_w"]
	payload["ars_h"] = prof["ars_h"]
	payload["rs_w"] = prof["rs_w"]
	payload["rs_h"] = prof["rs_h"]
	payload["rs_cd"] = prof["rs_cd"]
	payload["cg_w"] = prof["cg_w"]
	payload["cg_h"] = prof["cg_h"]
	payload["sg_w"] = prof["sg_w"]
	payload["sg_h"] = prof["sg_h"]
	payload["pr"] = prof["pr"]
	payload["so"] = prof["so"]
	payload["trrd"] = trrd
	payload["ucdv"] = false
	payload["dp0"] = false
	payload["hcovdr"] = false
	payload["plovdr"] = false
	payload["ftsovdr"] = false
	payload["orf"] = ""
	payload["dffls"] = dffls
	payload["niet"] = prof["niet"]
	payload["nid"] = prof["nid"]
	payload["nisd"] = prof["nisd"]
	for k, v := range navTiming {
		payload[k] = v
	}
	payload["lg"] = prof["lg"]
	payload["isb"] = false
	payload["idp"] = true
	payload["crt"] = 0
	payload["vnd"] = prof["vnd"]
	payload["bid"] = prof["bid"]
	payload["med"] = prof["med"]
	payload["pltod"] = false
	payload["wdifrm"] = false
	payload["npmtm"] = false
	payload["wdif"] = false
	payload["ccsT"] = errorStacks["ccsT"]
	payload["ccsB"] = errorStacks["ccsB"]
	payload["ccsH"] = errorStacks["ccsH"]
	payload["ccsV"] = errorStacks["ccsV"]
	payload["mmt"] = prof["mmt"]
	payload["wdifpnh"] = wdifpnh
	payload["vco"] = ""
	payload["vcots"] = false
	for k, v := range chromeVideoCodecs {
		payload[k] = v
	}
	payload["cssS"] = prof["cssS"]
	payload["css0"] = prof["css0"]
	payload["css1"] = prof["css1"]
	payload["cssH"] = prof["cssH"]
	payload["csssp"] = prof["csssp"]
	payload["muev"] = false
	for k, v := range chromeFeatures {
		payload[k] = v
	}
	payload["bchk"] = bchk
	payload["tz"] = prof["tz"]
	payload["ihdn"] = false
	payload["cdhf"] = false
	payload["eva"] = prof["eva"]
	payload["cokys"] = prof["cokys"]
	payload["ecpc"] = false
	payload["wop"] = false
	payload["pf"] = prof["pf"]
	payload["hc"] = prof["hc"]
	payload["br_oh"] = prof["br_oh"]
	payload["br_ow"] = prof["br_ow"]
	payload["ua"] = prof["ua"]
	payload["wbd"] = prof["wbd"]
	payload["ts_mtp"] = prof["ts_mtp"]
	payload["mob"] = prof["mob"]
	payload["lgs"] = prof["lgs"]
	payload["dvm"] = prof["dvm"]
	payload["ckwa"] = true
	for k, v := range chromeAudioCodecs {
		payload[k] = v
	}
	payload["ocpt"] = false
	payload["mq"] = "aptr:fine, ahvr:hover"
	payload["mq2"] = "cg:srgb, dr:standard, dm:browser"
	for k, v := range chromeBotChecks {
		payload[k] = v
	}
	payload["nhi"] = prof["nhi"]
	payload["k_lyts"] = prof["k_lyts"]
	payload["k_lytk"] = prof["k_lytk"]
	payload["bci"] = true
	payload["bcl"] = 1
	payload["bct"] = 0
	payload["bdt"] = nil
	payload["stqe"] = prof["stqe"]
	payload["stqu"] = prof["stqu"]
	payload["isf"] = false
	payload["isf2"] = false
	payload["pw"] = false
	payload["pcb"] = false
	payload["arc"] = false
	payload["fai"] = false
	payload["gai"] = false
	payload["bbs3"] = false
	payload["dt"] = true
	payload["fph"] = computeFph(prof)
	checksums := computeSignalChecksums(payload)
	payload["sgb"] = checksums["sgb"]
	payload["sgd"] = checksums["sgd"]
	payload["sgc"] = checksums["sgc"]
	payload["jset"] = jset
	bpc := opts.BPC
	if bpc < 1 {
		bpc = 1
	}
	payload["bpc"] = bpc

	for k, v := range opts.Overrides {
		payload[k] = v
	}

	order := []string{
		"log2", "r3n", "glvd", "glrd", "nddc", "exp8",
		"plu", "plgod", "plg", "plgne", "plgre", "plgof", "plggt",
		"bfr", "hdn", "br_w", "br_h", "br_iw", "br_ih",
		"ars_w", "ars_h", "rs_w", "rs_h", "rs_cd",
		"cg_w", "cg_h", "sg_w", "sg_h", "pr", "so", "trrd",
		"ucdv", "dp0", "hcovdr", "plovdr", "ftsovdr",
		"orf", "dffls", "niet", "nid", "nisd",
		"nt_tcp", "nt_dns", "nt_rd", "nt_irt", "nt_rt", "nt_tls",
		"nt_ttf", "nt_swt", "nt_csd", "nt_nhp", "nt_rdc", "nt_it",
		"nt_prs", "nt_esc", "nt_ttrd", "nt_le", "nt_dcle", "nt_di", "nt_dc",
		"lg", "isb", "idp", "crt", "vnd", "bid", "med",
		"pltod", "wdifrm", "npmtm", "wdif",
		"ccsT", "ccsB", "ccsH", "ccsV", "mmt", "wdifpnh",
		"vco", "vcots", "vch", "vchts", "vcw", "vcwts", "vc3", "vc3ts",
		"vcmp", "vcmpts", "vc1", "vc1ts", "vcmk", "vcmkuts", "vcq", "vcqts",
		"cssS", "css0", "css1", "cssH", "csssp", "muev",
		"pro_t", "wglo", "prso", "wbst", "psn", "edp", "addt", "wsdc",
		"ccsr", "nuad", "bcda", "idn", "capi", "svde", "vpbq",
		"bchk", "tz", "ihdn", "cdhf", "eva", "cokys", "ecpc", "wop",
		"pf", "hc", "br_oh", "br_ow", "ua", "wbd", "ts_mtp", "mob", "lgs", "dvm",
		"ckwa",
		"aco", "acots", "acmp", "acmpts", "acmpu", "acmputs", "acw", "acwts",
		"acma", "acmats", "acaa", "acaats", "ac3", "ac3ts", "acf", "acfts",
		"acmp4", "acmp4ts", "acmp3", "acmp3ts", "acwm", "acwmts",
		"ocpt", "mq", "mq2",
		"awe", "phe", "dat", "nm", "geb", "sqt", "spwn", "emt",
		"nhi", "k_lyts", "k_lytk",
		"bci", "bcl", "bct", "bdt",
		"stqe", "stqu", "isf", "isf2",
		"pw", "pcb", "arc", "fai", "gai", "bbs3", "dt",
		"fph", "sgb", "sgd", "sgc", "jset", "bpc",
	}

	signals := make([]ddcrypto.Signal, 0, len(order))
	seen := make(map[string]bool)
	for _, key := range order {
		if v, ok := payload[key]; ok {
			signals = append(signals, ddcrypto.Signal{Key: key, Value: v})
			seen[key] = true
		}
	}
	for key, v := range payload {
		if !seen[key] {
			signals = append(signals, ddcrypto.Signal{Key: key, Value: v})
		}
	}
	return signals
}
