"use strict";

const VuePlayer = {
	template: '#mp3-player-tpl',
	data() {
		return {
			visible: false,
			list: [],
			selfile: {},
			volval: 100,
			ratval: 6,
			repeatmode: 0, // 0 - no any repeat, 1 - repeat single, 2 - repeat playlist
			audioonly: false,
			seeking: false,
			media: null,
			autoplay: false,
			ready: false,
			timecur: 0,
			timebuf: 0,
			timeend: 0,
			ratevals: [ // rate predefined values
				1 / 2.50, 1 / 2.00, 1 / 1.75, 1 / 1.50, 1 / 1.25, 1 /1.15, 1, 1.15, 1.25, 1.50, 1.75, 2.00, 2.50
			],

			iid: makestrid(10) // instance ID
		};
	},
	computed: {
		// list of files that can be played
		playlist() {
			const l = [];
			for (const file of this.list) {
				if (audiofilter(file) || !this.audioonly && videofilter(file)) {
					l.push(file);
				}
			}
			return l;
		},
		// index of selected file
		selfilepos() {
			for (const i in this.playlist) {
				if (this.selfile === this.playlist[i]) {
					return Number(i);
				}
			}
		},
		// returns previous file in list
		getprev() {
			return this.selfile && this.selfilepos > 0
				? this.playlist[this.selfilepos - 1]
				: this.repeatmode === 2 && this.playlist[this.playlist.length - 1];
		},
		// returns next file in list
		getnext() {
			return this.selfile && this.selfilepos < this.playlist.length - 1
				? this.playlist[this.selfilepos + 1]
				: this.repeatmode === 2 && this.playlist[0];
		},

		volumelabel() {
			return this.volval;
		},
		ratelabel() {
			return this.ratevals[this.ratval].toFixed(2);
		},

		// audio buttons
		iconplay() {
			return this.selfile.playback ? 'pause' : 'play_arrow';
		},
		hintplay() {
			return this.selfile.playback ? 'pause' : 'play';
		},
		iconrepeat() {
			return this.repeatmode === 1 ? 'repeat_one' : 'repeat';
		},
		hintrepeat() {
			switch (this.repeatmode) {
				case 0: return "no any repeat";
				case 1: return "repeat one";
				case 2: return "repeat playlist";
			}
		},
		// progress bar
		fmttimecur() {
			return fmttime(this.timecur, this.timeend);
		},
		fmttimeend() {
			return fmttime(this.timeend, this.timeend);
		},
		fmttrackinfo() {
			if (this.selfile.title) {
				return `${this.selfile.artist ?? this.selfile.album ?? this.selfile.genre ?? ''} - ${this.selfile.title}`;
			} else {
				return this.selfile.name;
			}
		},
		stlbarcur() {
			const percent = this.timeend === Infinity ? 95 : // streamed
				!this.timeend || Number.isNaN(this.timeend) ? 5 : // unknown length
					this.timecur / this.timeend * 100;
			return { width: percent + "%" };
		},
		stlbarbuf() {
			const percent = Number.isFinite(this.timeend)
				? this.timebuf / this.timeend * 100 : 0;
			return { width: percent + "%" };
		}
	},
	methods: {
		setup(file) {
			if (this.selfile.puid === file.puid) { // do not set again same file
				return;
			}
			this.selfile = file;

			const media = new Audio(mediaurl(file, 1, 0)); // API HTMLMediaElement, HTMLAudioElement
			media.volume = this.volval / 100;
			media.playbackRate = this.ratevals[this.ratval];
			media.loop = this.repeatmode === 1;
			media.autoplay = this.autoplay;

			// reassign media current content
			if (this.media && !this.media.paused) {
				this.media.pause();
			}
			this.media = media;

			// disable UI for not ready media
			this.ready = false;

			const updateprogress = () => {
				if (!this.seeking) {
					this.timecur = media.currentTime;
				}

				if (media.buffered.length > 0) {
					const cur = this.timecur;
					const pos1 = media.buffered.start(0);
					const pos2 = media.buffered.end(0);
					if (pos1 <= cur && pos2 > cur) { // buffered in current pos
						this.timebuf = pos2 - cur;
					} else { // not buffered or buffered outside
						this.timebuf = 0;
					}
				} else {
					this.timebuf = 0;
				}
			};

			// media interface responders
			media.addEventListener('loadedmetadata', () => {
				this.timecur = media.currentTime;
				this.timebuf = 0;
				this.timeend = media.duration;

				updateprogress();
			});
			media.addEventListener('canplay', () => {
				// enable UI
				this.ready = true;
				// load to player
				if (!media.autoplay) {
					media.play();
					media.pause();
				}
			});
			media.addEventListener('timeupdate', updateprogress);
			media.addEventListener('seeked', updateprogress);
			media.addEventListener('progress', updateprogress);
			media.addEventListener('durationchange', updateprogress);
			media.addEventListener('play', () => {
				this.autoplay = true;
				media.autoplay = true;
				eventHub.emit('playback', file, true);
			});
			media.addEventListener('pause', () => {
				this.autoplay = false;
				media.autoplay = false;
				eventHub.emit('playback', file, false);
			});
			media.addEventListener('ended', () => {
				this.autoplay = true;
				this.onnext();
			});
			media.addEventListener('error', e => {
				if (e.message) {
					console.error("Error " + e.code + "; details: " + e.message);
				} else {
					console.error(e);
				}
			});
			media.addEventListener('volumechange', () => {
				this.volval = media.volume * 100;
			});
			media.addEventListener('ratechange', () => {
				this.ratval = (() => {
					const r = media.playbackRate;
					let pp = 1 / 3.0, pi = this.ratevals[0];
					for (let i = 0; i < this.ratevals.length - 1; i++) {
						let pn = this.ratevals[i + 1];
						if (r >= (pi + pp) / 2 && r < (pi + pn) / 2) {
							return i;
						}
						pp = pi, pi = pn;
					}
					return this.ratevals.length - 1;
				})();
			});
		},
		popup(file) {
			this.visible = true;
			this.setup(file);
		},
		close() {
			this.visible = false;
			if (this.media && !this.media.paused) {
				this.media.pause();
				return true;
			}
			return false;
		},

		play() {
			if (this.media) {
				if (this.media.paused) {
					this.media.play();
				} else {
					this.media.pause();
				}
			}
		},

		// user events responders

		onprev() {
			if (this.getprev) {
				eventHub.emit('select', this.getprev);
			}
		},
		onplay() {
			this.play();
		},
		onnext() {
			if (this.getnext) {
				eventHub.emit('select', this.getnext);
			}
		},
		onrepeat() {
			this.repeatmode = (this.repeatmode + 1) % (this.list ? 3 : 2);
			if (this.media) {
				this.media.loop = this.repeatmode === 1;
			}
		},

		onseekerchange(e) {
			this.media.currentTime = e.target.value;
			this.seeking = false;
		},
		onseekerinput(e) {
			this.seeking = true;
			this.timecur = Number(e.target.value);
		},

		onvolinp(e) {
			this.volval = Number(e.target.value);
		},
		onvolval(e) {
			if (this.media) {
				this.media.volume = this.volval / 100;
			}
		},
		onratinp(e) {
			this.ratval = Number(e.target.value);
		},
		onratval(e) {
			if (this.media) {
				this.media.playbackRate = this.ratevals[this.ratval];
			}
		},

		onopen(file) {
			if (file.type || !file.size) {
				return;
			}
			if (audiofilter(file) || videofilter(file)) {
				if (this.media && !this.media.paused) {
					this.media.pause();
				}
			}
		},
		onselect(file) {
			const is = file => file && (audiofilter(file) || !this.audioonly && videofilter(file));
			if (this.visible) {
				if (is(file)) {
					this.setup(file);
				} else {
					this.close();
				}
			} else if (is(file)) {
				this.popup(file);
			}
		},
		onplaylist(list) {
			this.list = list;
		},
		onaudioonly(val) {
			this.audioonly = val;
		}
	},
	created() {
		eventHub.on('open', this.onopen);
		eventHub.on('select', this.onselect);
		eventHub.on('playlist', this.onplaylist);
		eventHub.on('audioonly', this.onaudioonly);
	},
	unmounted() {
		eventHub.off('open', this.onopen);
		eventHub.off('select', this.onselect);
		eventHub.off('playlist', this.onplaylist);
		eventHub.off('audioonly', this.onaudioonly);
	}
};

// The End.
