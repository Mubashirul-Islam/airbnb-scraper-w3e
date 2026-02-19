package scraper

// CSS selectors used across the scraper.
// Centralising them makes future updates trivial.
const (
	// Search results page
	PropertyCardSelector = `.c965t3n.atm_9s_11p5wf0.atm_dz_1osqo2v.dir.dir-ltr`
	CardContainerFallback = `[data-testid="card-container"], [itemprop="itemListElement"], .cy5jw6o`

	// Pagination
	NextPageSelector = `.l1ovpqvx.atm_npmupv_14b5rvc_10saat9.atm_4s4swg_18xq13z_10saat9` +
		`.atm_u9em2p_1r3889l_10saat9.atm_1ezpcqw_1u41vd9_10saat9.atm_fyjbsv_c4n71i_10saat9` +
		`.atm_1rna0z7_1uk391_10saat9.c1ytbx3a.atm_mk_h2mmj6.atm_9s_1txwivl.atm_h_1h6ojuz` +
		`.atm_fc_1h6ojuz.atm_bb_idpfg4.atm_26_1j28jx2.atm_3f_glywfm.atm_7l_b0j8a8` +
		`.atm_gi_idpfg4.atm_l8_idpfg4.atm_uc_oxy5qq.atm_kd_glywfm.atm_gz_opxopj` +
		`.atm_uc_glywfm__1rrf6b5.atm_26_ppd4by_1rqz0hn_uv4tnr.atm_tr_kv3y6q_csw3t1` +
		`.atm_26_ppd4by_1ul2smo.atm_3f_glywfm_jo46a5.atm_l8_idpfg4_jo46a5` +
		`.atm_gi_idpfg4_jo46a5.atm_3f_glywfm_1icshfk.atm_kd_glywfm_19774hq` +
		`.atm_70_glywfm_1w3cfyq.atm_uc_1wx0j5_9xuho3.atm_70_1j5h5ka_9xuho3` +
		`.atm_26_ppd4by_9xuho3.atm_uc_glywfm_9xuho3_1rrf6b5.atm_7l_156bl0x_1o5j5ji` +
		`.atm_9j_13gfvf7_1o5j5ji.atm_26_1j28jx2_154oz7f.atm_92_1yyfdc7_vmtskl` +
		`.atm_9s_1ulexfb_vmtskl.atm_mk_stnw88_vmtskl.atm_tk_1ssbidh_vmtskl` +
		`.atm_fq_1ssbidh_vmtskl.atm_tr_pryxvc_vmtskl.atm_vy_1vi7ecw_vmtskl` +
		`.atm_e2_1vi7ecw_vmtskl.atm_5j_1ssbidh_vmtskl.atm_mk_h2mmj6_1ko0jae.dir.dir-ltr`

	// Detail page
	DetailReadySelector = `h1, [data-section-id="OVERVIEW_DEFAULT"]`
	PriceSelector       = `span.u1opajno, span.u174bpcy`

	// Detail page extraction (JS selectors)
	JSPriceSelector   = `span.u1opajno, span.u174bpcy`
	JSRatingSelector  = `div[data-testid="pdp-reviews-highlight-banner-host-rating"] div[aria-hidden="true"], .r1lcxetl.atm_c8_o7aogt.atm_c8_l52nlx__oggzyc`
	JSDescSelector    = `span .l1h825yc.atm_kd_adww2_24z95b`
)
