#!/bin/bash -u
# This script downloads all javascript dependencies used on frontend.

plugdir=$(realpath -s "$(dirname $0)/../frontend/plugin")
mkdir -pv "$plugdir/images"

# bootstrap 5.2.3
# https://cdnjs.com/libraries/bootstrap
curl https://cdnjs.cloudflare.com/ajax/libs/bootstrap/5.2.3/js/bootstrap.min.js --output $plugdir/bootstrap.min.js
curl https://cdnjs.cloudflare.com/ajax/libs/bootstrap/5.2.3/js/bootstrap.min.js.map --output $plugdir/bootstrap.min.js.map
curl https://cdnjs.cloudflare.com/ajax/libs/bootstrap/5.2.3/css/bootstrap.min.css --output $plugdir/bootstrap.min.css
curl https://cdnjs.cloudflare.com/ajax/libs/bootstrap/5.2.3/css/bootstrap.min.css.map --output $plugdir/bootstrap.min.css.map

# popper 2.11.6
# https://cdnjs.com/libraries/popper.js
curl https://cdnjs.cloudflare.com/ajax/libs/popper.js/2.11.6/umd/popper.min.js --output $plugdir/popper.min.js
curl https://cdnjs.cloudflare.com/ajax/libs/popper.js/2.11.6/umd/popper.min.js.map --output $plugdir/popper.min.js.map

# Vue 3.2.45
# https://cdnjs.com/libraries/vue
# https://unpkg.com/vue@next
curl https://unpkg.com/vue@3.2.45/dist/vue.global.js --output $plugdir/vue.global.js
curl https://unpkg.com/vue@3.2.45/dist/vue.global.prod.js --output $plugdir/vue.global.prod.js

# leaflet 1.9.3
# https://cdnjs.com/libraries/leaflet
curl https://cdnjs.cloudflare.com/ajax/libs/leaflet/1.9.3/leaflet.js --output $plugdir/leaflet.js
curl https://cdnjs.cloudflare.com/ajax/libs/leaflet/1.9.3/leaflet.js.map --output $plugdir/leaflet.js.map
curl https://cdnjs.cloudflare.com/ajax/libs/leaflet/1.9.3/leaflet.min.css --output $plugdir/leaflet.min.css
curl https://cdnjs.cloudflare.com/ajax/libs/leaflet/1.9.3/images/layers.png --output $plugdir/images/layers.png
curl https://cdnjs.cloudflare.com/ajax/libs/leaflet/1.9.3/images/layers-2x.png --output $plugdir/images/layers-2x.png
curl https://cdnjs.cloudflare.com/ajax/libs/leaflet/1.9.3/images/marker-icon.png --output $plugdir/images/marker-icon.png
curl https://cdnjs.cloudflare.com/ajax/libs/leaflet/1.9.3/images/marker-icon-2x.png --output $plugdir/images/marker-icon-2x.png
curl https://cdnjs.cloudflare.com/ajax/libs/leaflet/1.9.3/images/marker-shadow.png --output $plugdir/images/marker-shadow.png

# MarkerCluster 1.5.3
# https://cdnjs.com/libraries/leaflet.markercluster
curl https://cdnjs.cloudflare.com/ajax/libs/leaflet.markercluster/1.5.3/leaflet.markercluster.js --output $plugdir/leaflet.markercluster.js
curl https://cdnjs.cloudflare.com/ajax/libs/leaflet.markercluster/1.5.3/leaflet.markercluster.js.map --output $plugdir/leaflet.markercluster.js.map
curl https://cdnjs.cloudflare.com/ajax/libs/leaflet.markercluster/1.5.3/MarkerCluster.css --output $plugdir/MarkerCluster.css
curl https://cdnjs.cloudflare.com/ajax/libs/leaflet.markercluster/1.5.3/MarkerCluster.Default.css --output $plugdir/MarkerCluster.Default.css

# sha256 0.9.0
# https://cdnjs.com/libraries/js-sha256
curl https://cdnjs.cloudflare.com/ajax/libs/js-sha256/0.9.0/sha256.min.js --output $plugdir/sha256.min.js

# normalize 8.0.1
# https://cdnjs.com/libraries/normalize
curl https://cdnjs.cloudflare.com/ajax/libs/normalize/8.0.1/normalize.min.css --output $plugdir/normalize.min.css
curl https://cdnjs.cloudflare.com/ajax/libs/normalize/8.0.1/normalize.min.css.map --output $plugdir/normalize.min.css.map
