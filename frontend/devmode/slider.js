"use strict";

const controlstimeout = 2500; // timeout in milliseconds
const playlisttimeout = 8000; // timeout in milliseconds

// set to true after touch
let touched = false;

const VueThumbSlider = {
	template: '#thumbslider-tpl',
	props: ['list'],
	data() {
		return {
			selfile: null
		};
	},
	methods: {
		onwheel(e) {
			this.$refs.slide.scrollBy({ left: e.deltaX + e.deltaY, behavior: 'smooth' });
		},

		onprev() {
			this.$refs.slide.scrollBy({ left: -125, behavior: 'smooth' });
		},
		onnext() {
			this.$refs.slide.scrollBy({ left: +125, behavior: 'smooth' });
		},

		onselect(file) {
			this.selfile = file;
		},
		oniconclick(file) {
			eventHub.emit('select', file);
		}
	},
	created() {
		eventHub.on('select', this.onselect);
	},
	unmounted() {
		eventHub.off('select', this.onselect);
	}
};

const VuePhotoSlider = {
	template: '#photoslider-tpl',
	data() {
		return {
			loadbar: false,
			list: [],
			autolist: false,
			hd: true,
			selfile: null,
			repeatmode: 0, // 0 - no any repeat, 1 - repeat single, 2 - repeat playlist
			ctrlhnd: 0,
			alhnd: 0,
			dlg: null
		};
	},
	computed: {
		isimage() {
			return this.selfile && imagefilter(this.selfile);
		},
		isvideo() {
			return this.selfile && videofilter(this.selfile);
		},
		// list of visible by this viewer files
		viewlist() {
			const l = [];
			for (const file of this.list) {
				if (imagefilter(file) || videofilter(file)) {
					l.push(file);
				}
			}
			return l;
		},
		// image url
		selfileurl() {
			return this.selfile && `/id${this.$root.aid}/file/${this.selfile}?media=1&hd=${this.hd}`;
		},
		// index of selected file
		selfilepos() {
			for (const i in this.viewlist) {
				if (this.selfile === this.viewlist[i]) {
					return Number(i);
				}
			}
		},
		// returns previous file in list
		getprev() {
			return this.selfile && this.selfilepos > 0
				? this.viewlist[this.selfilepos - 1]
				: this.repeatmode === 2 && this.viewlist[this.viewlist.length - 1];
		},
		// returns next file in list
		getnext() {
			return this.selfile && this.selfilepos < this.viewlist.length - 1
				? this.viewlist[this.selfilepos + 1]
				: this.repeatmode === 2 && this.viewlist[0];
		},
		islist() {
			return this.viewlist.length > 1;
		}
	},
	methods: {
		isvisible() {
			return this.$el.offsetWidth > 0 && this.$el.offsetHeight > 0;
		},
		load(file) {
			if (this.selfile !== file) {
				if (!this.loadbar) { // check for new image loading during previous not loaded
					eventHub.emit('ajax', +1);
					this.loadbar = true;
				}

				// remove previous autolist timer
				if (this.alhnd) {
					clearTimeout(this.alhnd);
				}

				if (this.isvideo && !this.$refs.video.paused) {
					this.$refs.video.pause();
				}
				this.selfile = file;
				if (this.isvideo) {
					this.$refs.video.src = this.selfileurl
				}
			}
		},
		popup(file, list) {
			if (isFullscreen()) {
				closeFullscreen();
			}
			this.list = list ?? [file];
			this.load(file);
			this.dlg.show();
			this.showcontrols();
		},
		close() {
			this.dlg.hide();
		},
		showcontrols() {
			// remove previous controls timer
			if (this.ctrlhnd) {
				clearTimeout(this.ctrlhnd);
			}
			// set new timer to hide controls
			this.ctrlhnd = setTimeout(() => {
				this.ctrlhnd = 0;
			}, controlstimeout);
		},
		hidecontrols() {
			if (this.ctrlhnd) {
				clearTimeout(this.ctrlhnd);
				this.ctrlhnd = 0;
			}
		},

		onimgload(e) {
			eventHub.emit('ajax', -1);
			this.loadbar = false;

			// set new autolist timer
			if (this.isimage) {
				this.alhnd = setTimeout(() => {
					this.alhnd = 0;
					if (this.autolist) {
						this.onnext();
					}
				}, playlisttimeout);
			}
		},
		onimgerror(e) {
			eventHub.emit('ajax', -1);
			this.loadbar = false;
		},
		onended(e) {
			if (this.autolist) {
				this.onnext();
			}
		},
		ontouchstart(e) {
			touched = true;
		},
		onclick(e) {
			touched = false;
			// remove autolist timer
			if (this.alhnd) {
				clearTimeout(this.alhnd);
				this.alhnd = 0;
			}
			if (this.ctrlhnd) {
				// remove previous controls timer
				clearTimeout(this.ctrlhnd);
				this.ctrlhnd = 0;
			} else {
				// set new timer to hide controls
				this.ctrlhnd = setTimeout(() => {
					this.ctrlhnd = 0;
				}, controlstimeout);
			}
		},
		onmove(e) {
			if (!touched) {
				this.showcontrols();
			}
		},
		onkeyup(e) {
			switch (e.code) {
				case 'ArrowLeft':
					this.onprev();
					break;
				case 'ArrowRight':
					this.onnext();
					break;
				case 'Space':
					this.onclick(e)
					break;
				case 'Home':
					if (this.viewlist.length) {
						eventHub.emit('select', this.viewlist[0]);
					}
					break;
				case 'End':
					if (this.viewlist.length) {
						eventHub.emit('select', this.viewlist[this.viewlist.length - 1]);
					}
					break;
				default:
					this.showcontrols();
			}
		},
		onshowcontrols() {
			this.showcontrols();
		},
		onprev() {
			if (this.getprev) {
				eventHub.emit('select', this.getprev);
			}
		},
		onnext() {
			if (this.getnext) {
				eventHub.emit('select', this.getnext);
			}
		},
		onclose() {
			this.close();
		},

		onopen(file, list) {
			if (imagefilter(file) || videofilter(file)) {
				this.popup(file, list);
			}
		},
		onselect(file) {
			if (this.isvisible()) {
				if (file && (imagefilter(file) || videofilter(file))) {
					this.load(file);
					// update controls timer if it set
					if (this.ctrlhnd) {
						clearTimeout(this.ctrlhnd);
						this.ctrlhnd = setTimeout(() => {
							this.ctrlhnd = 0;
						}, controlstimeout);
					}
				} else {
					this.close();
				}
			}
		}
	},
	mounted() {
		eventHub.on('open', this.onopen);
		eventHub.on('select', this.onselect);

		this.dlg = new bootstrap.Modal(this.$el);
		this.$el.addEventListener('shown.bs.modal', e => {
		});
		this.$el.addEventListener('hidden.bs.modal', e => {
			if (this.isvideo && !this.$refs.video.paused) {
				this.$refs.video.pause();
			}
			this.selfile = null;
		});
	},
	unmounted() {
		eventHub.off('open', this.onopen);
		eventHub.off('select', this.onselect);
		this.dlg = null;
	}
};

// The End.
