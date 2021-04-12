"use strict";

const isTypeJPEG = ext => ({
	".jpg": true, ".jpe": true, ".jpeg": true, ".jfif": true
})[ext];

const isMainImage = ext => ({
	".tga": true, ".bmp": true, ".dib": true, ".rle": true, ".dds": true,
	".tif": true, ".tiff": true, ".jpg": true, ".jpe": true, ".jpeg": true, ".jfif": true,
	".gif": true, ".png": true, ".webp": true, ".psd": true, ".psb": true
})[ext];

const photofilter = file => file.size && isMainImage(pathext(file.name));

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
		imgicon(file) {
			return filetmbimg(file);
		},

		onprev() {
			this.$refs.slide.scrollBy({ left: -120, behavior: 'smooth' });
		},
		onnext() {
			this.$refs.slide.scrollBy({ left: +120, behavior: 'smooth' });
		},
		onthumb(file) {
			this.$emit('select', file);
		}
	}
});

Vue.component('photoslider-tag', {
	template: '#photoslider-tpl',
	props: ["list"],
	data: function () {
		return {
			selfile: null,
			imgloading: false,
			repeatmode: 0 // 0 - no any repeat, 1 - repeat single, 2 - repeat playlist
		};
	},
	computed: {
		// image url
		selfileurl() {
			return this.selfile && mediaurl(this.selfile);
		},
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
					if (photofilter(file)) {
						return file;
					}
				}
			};
			return this.selfile && (prevpos(this.selfilepos, -1) || this.repeatmode === 2 && prevpos(this.list.length, this.selfilepos));
		},
		// returns next file in list
		getnext() {
			const nextpos = (from, to) => {
				for (let i = from + 1; i < to; i++) {
					const file = this.list[i];
					if (photofilter(file)) {
						return file;
					}
				}
			};
			return this.selfile && (nextpos(this.selfilepos, this.list.length) || this.repeatmode === 2 && nextpos(-1, this.selfilepos));
		},
		islist() {
			return this.list && this.list.length > 1;
		}
	},
	methods: {
		load(file) {
			if (this.selfile !== file) {
				if (file) {
					this.selfile = file;
					this.imgloading = true;
				} else {
					this.selfile = null;
					this.imgloading = false;
				}
			}
		},
		select(file) {
			this.load(file);
			this.$emit('select', file);
		},
		popup(file) {
			this.load(file);
			$(this.$refs.modal).modal('show');
		},
		close() {
			this.load(null);
			$(this.$refs.modal).modal('hide');
		},

		onimgload(e) {
			this.imgloading = false;
		},
		onimgerror(e) {
			this.imgloading = false;
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
					if (this.list) {
						this.select(this.list[0]);
					}
					break;
				case 'End':
					if (this.list) {
						this.select(this.list[this.list.length - 1]);
					}
					break;
			}
		},
		onselect(file) {
			this.select(file);
		},
		onprev() {
			if (this.getprev) {
				this.select(this.getprev);
			}
		},
		onnext() {
			if (this.getnext) {
				this.select(this.getnext);
			}
		},
		onclose() {
			this.close();
		}
	}
});

// The End.
