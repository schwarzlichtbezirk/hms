"use strict";

Vue.component('mp3-player-tag', {
	template: '#mp3-player-tpl',
	props: ["playlist"],
	data: function () {
		return {
			file: {},
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
		setup() {
			if (this.media && this.media.paused) {
				this.media.play();
				this.isplay = true;
				this.$emit('playback', this.file, true);
				return true;
			}
			return false;
		},

		close() {
			if (this.media && !this.media.paused) {
				this.media.pause();
				this.isplay = false;
				this.$emit('playback', this.file, false);
				return true;
			}
			this.isplay = false;
			this.$emit('playback', this.file, false);
			return false;
		},

		setfile(file) {
			if (this.file.path === file.path) { // do not set again same file
				return;
			}
			const playonshow = this.close();
			this.file = file;
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
					this.setup();
				}
			});
			this.media.addEventListener('timeupdate', () => this.updateprogress());
			this.media.addEventListener('seeked', () => this.updateprogress());
			this.media.addEventListener('progress', () => this.updateprogress());
			this.media.addEventListener('play', () => { });
			this.media.addEventListener('pause', () => { });
			this.media.addEventListener('ended', () => {
				const filepos = () => {
					for (const i in this.playlist) {
						const file = this.playlist[i];
						if (this.file.path === file.path) {
							return Number(i);
						}
					}
				};
				const nextpos = (pos) => {
					for (let i = pos + 1; i < this.playlist.length; i++) {
						const file = this.playlist[i];
						if (FTtoFV[file.type] === FV.music || FTtoFV[file.type] === FV.video) {
							return file;
						}
					}
				};
				const next1 = nextpos(filepos());
				if (next1) {
					this.$emit('select', next1);
					return;
				} else if (this.repeatmode === 2) {
					const next2 = nextpos(-1);
					if (next2) {
						this.$emit('select', next2);
						return;
					}
				}
				this.isplay = false;
				this.$emit('playback', this.file, false);
			});
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
				this.$emit('playback', this.file, true);
			} else {
				this.media.pause();
				this.isplay = false;
				this.isflowing = false;
				this.$emit('playback', this.file, false);
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
			const filepos = () => {
				for (const i in this.playlist) {
					const file = this.playlist[i];
					if (this.file.path === file.path) {
						return Number(i);
					}
				}
			};
			const prevpos = (pos) => {
				for (let i = pos - 1; i >= 0; i--) {
					const file = this.playlist[i];
					if (FTtoFV[file.type] === FV.music || FTtoFV[file.type] === FV.video) {
						return file;
					}
				}
			};
			const prev1 = prevpos(filepos());
			if (prev1) {
				this.$emit('select', prev1);
			} else {
				const prev2 = prevpos(this.playlist.length);
				if (prev2) {
					this.$emit('select', prev2);
				}
			}
		},

		onplay() {
			this.play();
		},

		onnext() {
			const filepos = () => {
				for (const i in this.playlist) {
					const file = this.playlist[i];
					if (this.file.path === file.path) {
						return Number(i);
					}
				}
			};
			const nextpos = (pos) => {
				for (let i = pos + 1; i < this.playlist.length; i++) {
					const file = this.playlist[i];
					if (FTtoFV[file.type] === FV.music || FTtoFV[file.type] === FV.video) {
						return file;
					}
				}
			};
			const next1 = nextpos(filepos());
			if (next1) {
				this.$emit('select', next1);
			} else {
				const next2 = nextpos(-1);
				if (next2) {
					this.$emit('select', next2);
				}
			}
		},

		onrepeat() {
			this.repeatmode = (this.repeatmode + 1) % 3;
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
