"use strict";

const mp3filter = file => FTtoFV[file.type] === FV.music || FTtoFV[file.type] === FV.video;

Vue.component('mp3-player-tag', {
	template: '#mp3-player-tpl',
	props: ["list"],
	data: function () {
		return {
			visible: false,
			selfile: {},
			rate: 1.00,
			volume: 1.00,
			repeatmode: 0, // 0 - no any repeat, 1 - repeat single, 2 - repeat playlist
			seeking: false,
			media: null,
			isplay: false, // this.media && !this.media.paused
			isflowing: false,
			ready: false,
			timecur: 0,
			timebuf: 0,
			timeend: 0,

			iid: makestrid(10) // instance ID
		};
	},
	computed: {
		// index of selected file
		selfilepos() {
			for (const i in this.list) {
				if (this.selfile.path === this.list[i].path) {
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
			return prevpos(this.selfilepos, -1) || this.repeatmode === 2 && prevpos(this.list.length, this.selfilepos);
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
			return nextpos(this.selfilepos, this.list.length) || this.repeatmode === 2 && nextpos(-1, this.selfilepos);
		},

		// music buttons
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
			if (this.selfile.path === file.path) { // do not set again same file
				return;
			}
			this.close();
			this.selfile = file;

			this.media = new Audio(getfileurl(file)); // API HTMLMediaElement, HTMLAudioElement
			this.media.playbackRate = this.rate;
			this.media.loop = this.repeatmode === 1;

			// disable UI for not ready media
			this.ready = false;

			// media interface responders
			this.media.addEventListener('loadedmetadata', () => {
				this.updateprogress();
			});
			this.media.addEventListener('canplay', () => {
				const cur = this.media.currentTime;
				const len = this.media.duration;
				this.timecur = cur;
				this.timebuf = 0;
				this.timeend = len;

				// enable UI
				this.ready = true;

				if (this.isflowing) {
					this.media.play();
					this.isplay = true;
					this.$emit('playback', this.selfile);
				}
			});
			this.media.addEventListener('timeupdate', () => this.updateprogress());
			this.media.addEventListener('seeked', () => this.updateprogress());
			this.media.addEventListener('progress', () => this.updateprogress());
			this.media.addEventListener('play', () => { });
			this.media.addEventListener('pause', () => { });
			this.media.addEventListener('ended', () => {
				this.isplay = false;
				this.$emit('playback', null);
				this.onnext();
			});
		},
		close() {
			if (this.media && !this.media.paused) {
				this.media.pause();
				this.isplay = false;
				this.$emit('playback', null);
				return true;
			}
			return false;
		},

		setrate(rate) {
			this.rate = rate;
			if (this.media) {
				this.media.playbackRate = rate;
			}
		},

		play() {
			if (!this.media) return;
			if (this.media.paused) {
				this.media.play();
				this.isplay = true;
				this.isflowing = true;
				this.$emit('playback', this.selfile);
			} else {
				this.media.pause();
				this.isplay = false;
				this.isflowing = false;
				this.$emit('playback', null);
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
				this.$emit('select', this.getprev);
			}
		},

		onplay() {
			this.play();
		},

		onnext() {
			if (this.getnext) {
				this.$emit('select', this.getnext);
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
		}
	}
});

// The End.
