"use strict";

Vue.component('mp3-player-tag', {
	template: `
<div class="navbar navbar-expand-sm w-100" id="music-footer">
	<ul class="navbar-nav flex-row">
		<li><button class="btn btn-link nav-link" v-on:click="onprev" title="skip to previous"><i class="material-icons">skip_previous</i></button></li>
		<li><button class="btn btn-link nav-link bg-secondary" v-on:click="onplay" v-bind:disabled="!ready" v-bind:title="hintplay"><i class="material-icons">{{iconplay}}</i></button></li>
		<li><button class="btn btn-link nav-link" v-on:click="onnext" title="skip to next"><i class="material-icons">skip_next</i></button></li>
		<li><button class="btn btn-link nav-link" v-on:click="onrepeat" v-bind:class="{active:repeatmode>0}" v-bind:title="hintrepeat"><i class="material-icons">{{iconrepeat}}</i></button></li>
	</ul>
	<button class="navbar-toggler" type="button" data-toggle="collapse" v-bind:data-target="'#nav'+iid">
		<span class="navbar-toggler-icon"></span>
	</button>
	<div class="collapse navbar-collapse" v-bind:id="'nav'+iid">
		<div class="timescale flex-grow-1">
			<div class="progress position-relative">
				<div class="current progress-bar" v-bind:style="stlbarcur" v-bind:aria-valuenow="timecur" aria-valuemin="0" v-bind:aria-valuemax="timeend"></div>
				<div class="buffer progress-bar" v-bind:style="stlbarbuf" v-bind:aria-valuenow="timebuf" aria-valuemin="0" v-bind:aria-valuemax="timeend"></div>
				<div class="timer justify-content-center align-self-center d-flex position-absolute w-100">{{fmttimecur}}&nbsp/&nbsp{{fmttimeend}}</div>
				<input class="seeker position-absolute w-100" min="0" type="range" v-bind:max="timeend" v-bind:value="timecur" v-bind:disabled="!ready" v-on:change="onseekerchange" v-on:input="onseekerinput">
			</div>
			<div>{{file.name}}</div>
		</div>
		<ul class="navbar-nav flex-row">
			<li class="btn-group dropup">
				<button class="btn btn-link nav-link dropdown-toggle" data-toggle="dropdown" aria-haspopup="true" aria-expanded="false" title="playback rate"><i class="material-icons">slow_motion_video</i></button>
				<ul class="dropdown-menu dropdown-menu-right" id="rate">
					<li class="dropdown-item" v-on:click="setrate(2.50)" v-bind:class="{active:rate===2.50}">&times;2.50</li>
					<li class="dropdown-item" v-on:click="setrate(2.00)" v-bind:class="{active:rate===2.00}">&times;2.00</li>
					<li class="dropdown-item" v-on:click="setrate(1.75)" v-bind:class="{active:rate===1.75}">&times;1.75</li>
					<li class="dropdown-item" v-on:click="setrate(1.50)" v-bind:class="{active:rate===1.50}">&times;1.50</li>
					<li class="dropdown-item" v-on:click="setrate(1.25)" v-bind:class="{active:rate===1.25}">&times;1.25</li>
					<li class="dropdown-item" v-on:click="setrate(1.15)" v-bind:class="{active:rate===1.15}">&times;1.15</li>
					<li class="dropdown-item" v-on:click="setrate(1.00)" v-bind:class="{active:rate===1.00}">normal</li>
					<li class="dropdown-item" v-on:click="setrate(0.90)" v-bind:class="{active:rate===0.90}">&times;0.90</li>
					<li class="dropdown-item" v-on:click="setrate(0.80)" v-bind:class="{active:rate===0.80}">&times;0.80</li>
					<li class="dropdown-item" v-on:click="setrate(0.70)" v-bind:class="{active:rate===0.70}">&times;0.70</li>
					<li class="dropdown-item" v-on:click="setrate(0.60)" v-bind:class="{active:rate===0.60}">&times;0.60</li>
					<li class="dropdown-item" v-on:click="setrate(0.50)" v-bind:class="{active:rate===0.50}">&times;0.50</li>
					<li class="dropdown-item" v-on:click="setrate(0.40)" v-bind:class="{active:rate===0.40}">&times;0.40</li>
				</ul>
			</li>
		</ul>
	</div>
</div>
`,
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
