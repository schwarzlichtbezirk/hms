#!/bin/bash
pldir=$(dirname $0)/../frontend/plugin/
mkdir -pv $pldir/images

# bootstrap 5.2.1
# https://cdnjs.com/libraries/bootstrap
curl https://cdnjs.cloudflare.com/ajax/libs/bootstrap/5.2.1/js/bootstrap.min.js --output $pldir/bootstrap.min.js
curl https://cdnjs.cloudflare.com/ajax/libs/bootstrap/5.2.1/js/bootstrap.min.js.map --output $pldir/bootstrap.min.js.map
curl https://cdnjs.cloudflare.com/ajax/libs/bootstrap/5.2.1/css/bootstrap.min.css --output $pldir/bootstrap.min.css
curl https://cdnjs.cloudflare.com/ajax/libs/bootstrap/5.2.1/css/bootstrap.min.css.map --output $pldir/bootstrap.min.css.map

# popper 2.11.6
# https://cdnjs.com/libraries/popper.js
curl https://cdnjs.cloudflare.com/ajax/libs/popper.js/2.11.6/umd/popper.min.js --output $pldir/popper.min.js
curl https://cdnjs.cloudflare.com/ajax/libs/popper.js/2.11.6/umd/popper.min.js.map --output $pldir/popper.min.js.map

# Vue 3.2.40
# https://cdnjs.com/libraries/vue
# https://unpkg.com/vue@next
curl https://unpkg.com/vue@3.2.40/dist/vue.global.js --output $pldir/vue.global.js
curl https://unpkg.com/vue@3.2.40/dist/vue.global.prod.js --output $pldir/vue.global.prod.js

# leaflet 1.9.1
# https://cdnjs.com/libraries/leaflet
curl https://cdnjs.cloudflare.com/ajax/libs/leaflet/1.9.1/leaflet.js --output $pldir/leaflet.js
curl https://cdnjs.cloudflare.com/ajax/libs/leaflet/1.9.1/leaflet.js.map --output $pldir/leaflet.js.map
curl https://cdnjs.cloudflare.com/ajax/libs/leaflet/1.9.1/leaflet.min.css --output $pldir/leaflet.min.css
curl https://cdnjs.cloudflare.com/ajax/libs/leaflet/1.9.1/images/layers.png --output $pldir/images/layers.png
curl https://cdnjs.cloudflare.com/ajax/libs/leaflet/1.9.1/images/layers-2x.png --output $pldir/images/layers-2x.png
curl https://cdnjs.cloudflare.com/ajax/libs/leaflet/1.9.1/images/marker-icon.png --output $pldir/images/marker-icon.png
curl https://cdnjs.cloudflare.com/ajax/libs/leaflet/1.9.1/images/marker-icon-2x.png --output $pldir/images/marker-icon-2x.png
curl https://cdnjs.cloudflare.com/ajax/libs/leaflet/1.9.1/images/marker-shadow.png --output $pldir/images/marker-shadow.png

# MarkerCluster 1.5.3
# https://cdnjs.com/libraries/leaflet.markercluster
curl https://cdnjs.cloudflare.com/ajax/libs/leaflet.markercluster/1.5.3/leaflet.markercluster.js --output $pldir/leaflet.markercluster.js
curl https://cdnjs.cloudflare.com/ajax/libs/leaflet.markercluster/1.5.3/leaflet.markercluster.js.map --output $pldir/leaflet.markercluster.js.map
curl https://cdnjs.cloudflare.com/ajax/libs/leaflet.markercluster/1.5.3/MarkerCluster.css --output $pldir/MarkerCluster.css
curl https://cdnjs.cloudflare.com/ajax/libs/leaflet.markercluster/1.5.3/MarkerCluster.Default.css --output $pldir/MarkerCluster.Default.css

# sha256 0.9.0
# https://cdnjs.com/libraries/js-sha256
curl https://cdnjs.cloudflare.com/ajax/libs/js-sha256/0.9.0/sha256.min.js --output $pldir/sha256.min.js

# normalize 8.0.1
# https://cdnjs.com/libraries/normalize
curl https://cdnjs.cloudflare.com/ajax/libs/normalize/8.0.1/normalize.min.css --output $pldir/normalize.min.css
curl https://cdnjs.cloudflare.com/ajax/libs/normalize/8.0.1/normalize.min.css.map --output $pldir/normalize.min.css.map
