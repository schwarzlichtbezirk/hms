"use strict";

Vue.component('mp3-player-tag', {
	template: `
<div class="navbar navbar-expand-sm w-100" id="music-footer">
	<ul class="navbar-nav flex-row">
		<li class="btn btn-link nav-link prev" v-on:click="onprev" title="skip to previous"><i class="material-icons">skip_previous</i></li>
		<li class="btn btn-link nav-link play bg-secondary disabled" v-on:click="onplay" v-bind:title="hintplay"><i class="material-icons">{{iconplay}}</i></li>
		<li class="btn btn-link nav-link next" v-on:click="onnext" title="skip to next"><i class="material-icons">skip_next</i></li>
		<li class="btn btn-link nav-link repeat" v-on:click="onrepeat" v-bind:class="{active:repeatmode>0}" v-bind:title="hintrepeat"><i class="material-icons">{{iconrepeat}}</i></li>
	</ul>
	<button class="navbar-toggler" type="button" data-toggle="collapse" data-target="#music-footer > .navbar-collapse">
		<span class="navbar-toggler-icon"></span>
	</button>
	<div class="collapse navbar-collapse">
		<div class="timescale flex-grow-1">
			<div class="progress position-relative">
				<div class="current progress-bar"></div>
				<div class="buffer progress-bar"></div>
				<div class="timer justify-content-center align-self-center d-flex position-absolute w-100"><span class="time-pos"></span>&nbsp/&nbsp<span class="time-end"></span></div>
				<input type="range" min="0" max="100" class="seeker position-absolute w-100">
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
	props: [],
	data: function () {
		return {
			file: {},
			rate: 1.00,
			volume: 1.00,
			repeatmode: 0, // 0 - no any repeat, 1 - repeat single, 2 - repeat playlist
			seeking: false,
			media: null,
			isplay: false, // this.media && !this.media.paused

			frame: null,
			ratemenu: null,
			curbar: null,
			bufbar: null,
			timer: null,
			seeker: null,

			iid: makestrid(10) // instance ID
		};
	},
	computed: {
		// music buttons
		iconplay() {
			return this.isplay ? 'pause' : 'play_arrow';
		},
		hintplay() {
			return this.isplay ? 'play' : 'pause';
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
		}
	},
	methods: {
		setup() {
			if (this.media && this.media.paused) {
				this.media.play();
				this.isplay = true;
				app.playbackmode = true;
				return true;
			}
			return false;
		},

		close() {
			if (this.media && !this.media.paused) {
				this.media.pause();
				this.isplay = false;
				app.playbackmode = false;
				return true;
			}
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
			this.frame.find(".play").addClass('disabled');
			this.seeker.prop('disabled', true);

			// media interface responders
			this.media.addEventListener('loadedmetadata', () => {
				this.updateprogress();
			});
			this.media.addEventListener('canplay', () => {
				const len = this.media.duration;
				const cur = this.media.currentTime;
				// enable UI
				this.frame.find(".play").removeClass('disabled');
				this.seeker.prop('disabled', false);
				this.seeker.attr('min', "0");
				this.seeker.attr('max', len.toString());
				this.seeker.val(cur.toString());
				this.frame.find(".timescale .time-end").text(fmttime(len, len));
				if (playonshow) {
					this.setup();
				}
			});
			this.media.addEventListener('timeupdate', () => this.updateprogress());
			this.media.addEventListener('seeked', () => this.updateprogress());
			this.media.addEventListener('progress', () => this.updateprogress());
			this.media.addEventListener('play', () => { });
			this.media.addEventListener('pause', () => { });
			this.media.addEventListener('ended', () => {
				const pls = app.playlist;
				const filepos = () => {
					for (const i in pls) {
						const file = pls[i];
						if (this.file.path === file.path) {
							return Number(i);
						}
					}
				};
				const nextpos = (pos) => {
					for (let i = pos + 1; i < pls.length; i++) {
						const file = pls[i];
						if (file.type === Wave || file.type === FLAC ||
							file.type === MP3 || file.type === OGG ||
							file.type === MP4 || file.type === WebM) {
							return file;
						}
					}
				};
				const next1 = nextpos(filepos());
				if (next1) {
					app.selected = next1;
					this.setfile(next1, true);
					return;
				} else if (this.repeatmode === 2) {
					const next2 = nextpos(-1);
					if (next2) {
						app.selected = next2;
						this.setfile(next2, true);
						return;
					}
				}
				this.isplay = false;
				app.playbackmode = false;
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
				app.playbackmode = true;
			} else {
				this.media.pause();
				this.isplay = false;
				app.playbackmode = false;
			}
		},

		updateprogress() {
			const len = this.media.duration;
			const cur = this.media.currentTime;
			{
				let percent;
				if (len === Infinity) { // streamed
					percent = 95;
				} else if (isNaN(len)) { // unknown length
					percent = 5;
				} else {
					percent = cur / len * 100;
				}
				this.curbar.css("width", percent + "%");
			}

			if (this.media.buffered.length > 0) {
				const pos1 = this.media.buffered.start(0);
				const pos2 = this.media.buffered.end(0);
				let percent;
				if (pos1 <= cur && pos2 - cur > 0) { // buffered in current pos
					percent = (pos2 - cur) / len * 100;
				} else { // not buffered or buffered outside
					percent = 0;
				}
				this.bufbar.css("width", percent + "%");
			}

			if (!this.seeking) {
				this.timer.text(fmttime(cur, len));
				this.seeker.val(cur.toString());
			}
		},

		// user events responders

		onprev() {
		},

		onplay() {
			this.play();
		},

		onnext() {
		},

		onrepeat() {
			this.repeatmode = (this.repeatmode + 1) % 3;
			if (this.media) {
				this.media.loop = this.repeatmode === 1;
			}
		}
	},
	mounted() {
		const frame = $("#music-footer");
		this.frame = frame;
		this.ratemenu = frame.find("#rate");
		this.curbar = frame.find(".timescale > .progress > .current");
		this.bufbar = frame.find(".timescale > .progress > .buffer");
		this.timer = frame.find(".timescale .time-pos");
		this.seeker = frame.find(".timescale > .progress > .seeker");

		this.seeker.on('change', () => {
			this.media.currentTime = Number(this.seeker.val());
			this.seeking = false;
		});
		this.seeker.on('input', () => {
			this.seeking = true;
			this.timer.text(fmttime(Number(this.seeker.val()), this.media.duration));
		});
	}
});

// The End.
