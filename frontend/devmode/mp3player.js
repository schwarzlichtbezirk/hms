"use strict";

const isMainAudio = ext => ({
	".wav": true, ".flac": true, ".mp3": true, ".ogg": true, ".opus": true,
	".acc": true, ".m4a": true, ".alac": true
})[ext];

const isMainVideo = ext => ({
	".mp4": true, ".webm": true
})[ext];

const mp3filter = file => !file.type && file.size && (isMainAudio(pathext(file.name)) || isMainVideo(pathext(file.name)));

Vue.component('mp3-player-tag', {
	template: '#mp3-player-tpl',
	props: ["list"],
	data: function () {
		return {
			visible: false,
			selfile: {},
			volval: 100,
			ratval: 6,
			repeatmode: 0, // 0 - no any repeat, 1 - repeat single, 2 - repeat playlist
			seeking: false,
			media: null,
			isplay: false, // this.media && !this.media.paused
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
		// index of selected file
		selfilepos() {
			for (const i in this.list) {
				if (this.selfile.puid === this.list[i].puid) {
					return Number(i);
				}
			}
		},
		// returns previous file in list
		getprev() {
			const prevpos = (from, to) => {
				for (let i = from - 1; i > to; i--) {
					const file = this.list[i];
					if (mp3filter(file)) {
						return file;
					}
				}
			};
			return prevpos(this.selfilepos, -1) || this.repeatmode === 2 && prevpos(this.list.length, this.selfilepos - 1);
		},
		// returns next file in list
		getnext() {
			const nextpos = (from, to) => {
				for (let i = from + 1; i < to; i++) {
					const file = this.list[i];
					if (mp3filter(file)) {
						return file;
					}
				}
			};
			return nextpos(this.selfilepos, this.list.length) || this.repeatmode === 2 && nextpos(-1, this.selfilepos + 1);
		},

		volumelabel() {
			return this.volval;
		},
		ratelabel() {
			return this.ratevals[this.ratval].toFixed(2);
		},

		// audio buttons
		iconplay() {
			return this.isplay ? 'pause' : 'play_arrow';
		},
		hintplay() {
			return this.isplay ? 'pause' : 'play';
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
				return `${this.selfile.artist || this.selfile.album || this.selfile.genre || ''} - ${this.selfile.title}`;
			} else {
				return this.selfile.name;
			}
		},
		stlbarcur() {
			const percent = this.timeend === Infinity ? 95 : // streamed
				!this.timeend || isNaN(this.timeend) ? 5 : // unknown length
					this.timecur / this.timeend * 100;
			return { width: percent + "%" };
		},
		stlbarbuf() {
			const percent = this.timeend === Infinity ? 0 : // streamed
				!this.timeend || isNaN(this.timeend) ? 0 : // unknown length
					this.timebuf / this.timeend * 100;
			return { width: percent + "%" };
		}
	},
	methods: {
		setup(file) {
			if (this.selfile.puid === file.puid) { // do not set again same file
				return;
			}
			this.close();
			this.selfile = file;

			this.media = new Audio(mediaurl(file)); // API HTMLMediaElement, HTMLAudioElement
			this.media.volume = this.volval / 100;
			this.media.playbackRate = this.ratevals[this.ratval];
			this.media.loop = this.repeatmode === 1;
			this.media.autoplay = this.autoplay;

			// disable UI for not ready media
			this.ready = false;

			// media interface responders
			this.media.addEventListener('loadedmetadata', () => {
				this.timecur = this.media.currentTime;
				this.timebuf = 0;
				this.timeend = this.media.duration;

				this.updateprogress();
			});
			this.media.addEventListener('canplay', () => {
				// enable UI
				this.ready = true;
				// load to player
				if (!this.media.autoplay) {
					this.media.play();
					this.media.pause();
				}
			});
			this.media.addEventListener('timeupdate', () => this.updateprogress());
			this.media.addEventListener('seeked', () => this.updateprogress());
			this.media.addEventListener('progress', () => this.updateprogress());
			this.media.addEventListener('durationchange', () => this.updateprogress());
			this.media.addEventListener('play', () => {
				this.isplay = true;
				this.autoplay = true;
				this.$emit('playback', this.selfile);
			});
			this.media.addEventListener('pause', () => {
				this.isplay = false;
				this.autoplay = false;
				this.$emit('playback', null);
			});
			this.media.addEventListener('ended', () => {
				this.autoplay = true;
				this.onnext();
			});
			this.media.addEventListener('error', e => {
				if (e.message) {
					console.error("Error " + e.code + "; details: " + e.message);
				} else {
					console.error(e);
				}
			});
			this.media.addEventListener('volumechange', () => {
				this.volval = this.media.volume * 100;
			});
			this.media.addEventListener('ratechange', () => {
				this.ratval = (() => {
					const r = this.media.playbackRate;
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
		close() {
			if (this.media && !this.media.paused) {
				this.media.pause();
				return true;
			}
			return false;
		},

		play() {
			if (!this.media) return;
			if (this.media.paused) {
				this.media.play();
			} else {
				this.media.pause();
			}
		},

		updateprogress() {
			const cur = this.media.currentTime;

			if (!this.seeking) {
				this.timecur = cur;
			}

			if (this.media.buffered.length > 0) {
				const pos1 = this.media.buffered.start(0);
				const pos2 = this.media.buffered.end(0);
				if (pos1 <= cur && pos2 > cur) { // buffered in current pos
					this.timebuf = pos2 - cur;
				} else { // not buffered or buffered outside
					this.timebuf = 0;
				}
			} else {
				this.timebuf = 0;
			}
		},

		// user events responders

		onprev() {
			if (this.getprev) {
				eventHub.$emit('select', this.getprev);
			}
		},

		onplay() {
			this.play();
		},

		onnext() {
			if (this.getnext) {
				eventHub.$emit('select', this.getnext);
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
		}
	}
});

// The End.
