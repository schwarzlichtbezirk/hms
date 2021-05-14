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
			eventHub.$emit('select', file);
		}
	}
});

Vue.component('photoslider-tag', {
	template: '#photoslider-tpl',
	data: function () {
		return {
			list: [],
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
		isvisible() {
			return this.$refs.modal.offsetWidth > 0 && this.$refs.modal.offsetHeight > 0;
		},
		load(file) {
			if (this.selfile !== file) {
				this.selfile = file;
				this.imgloading = true;
			}
		},
		popup(file, list) {
			this.list = list || [file];
			this.load(file);
			$(this.$refs.modal).modal('show');
		},
		close() {
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
						eventHub.$emit('select', this.list[0]);
					}
					break;
				case 'End':
					if (this.list) {
						eventHub.$emit('select', this.list[this.list.length - 1]);
					}
					break;
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
		$(this.$refs.modal).on('shown.bs.modal', e => {
		});
		$(this.$refs.modal).on('hidden.bs.modal', e => {
			this.selfile = null;
			this.imgloading = false;
		});
	},
	beforeDestroy() {
		eventHub.$off('select', this.onselect);
		$(this.$refs.modal).off('shown.bs.modal');
		$(this.$refs.modal).off('hidden.bs.modal');
	}
});

// The End.
