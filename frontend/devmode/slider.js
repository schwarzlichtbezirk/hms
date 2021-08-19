"use strict";

const isTypeJPEG = ext => ({
	".jpg": true, ".jpe": true, ".jpeg": true, ".jfif": true
})[ext];

const isMainImage = ext => ({
	".tga": true, ".bmp": true, ".dib": true, ".rle": true, ".dds": true,
	".tif": true, ".tiff": true, ".jpg": true, ".jpe": true, ".jpeg": true, ".jfif": true,
	".gif": true, ".png": true, ".webp": true, ".psd": true, ".psb": true
})[ext];

const photofilter = file => !file.type && file.size && isMainImage(pathext(file.name));

const controlstimeout = 3000; // timeout in milliseconds

Vue.component('thumbslider-tag', {
	template: '#thumbslider-tpl',
	props: ["selfile", "list"],
	computed: {
		slide() {
			const lst = [];
			for (const file of this.list) {
				if (photofilter(file)) {
					lst.push(file);
				}
			}
			return lst;
		}
	},
	methods: {
		onselect(file) {
			eventHub.$emit('select', file);
		},
		onwheel(e) {
			this.$refs.slide.scrollBy({ left: e.deltaX + e.deltaY, behavior: 'smooth' });
		},

		onprev() {
			this.$refs.slide.scrollBy({ left: -125, behavior: 'smooth' });
		},
		onnext() {
			this.$refs.slide.scrollBy({ left: +125, behavior: 'smooth' });
		}
	}
});

Vue.component('photoslider-tag', {
	template: '#photoslider-tpl',
	data: function () {
		return {
			list: [],
			hd: 1,
			selfile: null,
			repeatmode: 0, // 0 - no any repeat, 1 - repeat single, 2 - repeat playlist
			ctrlhnd: 0,
			dlg: null
		};
	},
	computed: {
		// list of visible by this viewer files
		viewlist() {
			const l = [];
			for (const file of this.list) {
				if (photofilter(file)) {
					l.push(file);
				}
			}
			return l;
		},
		// image url
		selfileurl() {
			return this.selfile && mediaurl(this.selfile, 1, this.hd);
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
				this.selfile = file;
				eventHub.$emit('ajax', +1);
			}
			this.showcontrols();
		},
		popup(file, list) {
			if (isFullscreen()) {
				closeFullscreen();
			}
			this.list = list || [file];
			this.load(file);
			this.dlg.show();
		},
		close() {
			this.dlg.hide();
		},
		showcontrols() {
			// remove previous timer
			if (this.ctrlhnd) {
				clearTimeout(this.ctrlhnd);
			}
			// set new timer to hide
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
			eventHub.$emit('ajax', -1);
		},
		onimgerror(e) {
			eventHub.$emit('ajax', -1);
		},
		onmove(e) {
			this.showcontrols();
		},
		onkeyup(e) {
			switch (e.code) {
				case 'ArrowLeft':
					this.onprev();
					break;
				case 'ArrowRight':
				case 'Space':
					this.onnext();
					break;
				case 'Home':
					if (this.viewlist.length) {
						eventHub.$emit('select', this.viewlist[0]);
					}
					break;
				case 'End':
					if (this.viewlist.length) {
						eventHub.$emit('select', this.viewlist[this.viewlist.length - 1]);
					}
					break;
				default:
					this.showcontrols();
			}
		},
		onprev() {
			if (this.getprev) {
				eventHub.$emit('select', this.getprev);
			}
		},
		onnext() {
			if (this.getnext) {
				eventHub.$emit('select', this.getnext);
			}
		},
		onclose() {
			this.close();
		},

		onselect(file) {
			if (this.isvisible()) {
				if (file && photofilter(file)) {
					this.load(file);
				} else {
					this.close();
				}
			}
		}
	},
	created() {
		eventHub.$on('select', this.onselect);
	},
	mounted() {
		this.dlg = new bootstrap.Modal(this.$el);
		this.$el.addEventListener('shown.bs.modal', e => {
		});
		this.$el.addEventListener('hidden.bs.modal', e => {
			this.selfile = null;
		});
	},
	beforeDestroy() {
		eventHub.$off('select', this.onselect);
		this.dlg = null;
	}
});

// The End.
