#define _USE_MATH_DEFINES
#include <math.h>

const double Rearth = 6371e3; // metres
const double πrad   = M_PI / 180;

// Haversine uses formula to calculate the great-circle distance between
// two points – that is, the shortest distance over the earth’s surface –
// giving an ‘as-the-crow-flies’ distance between the points (ignoring
// any hills they fly over, of course!).
//
// See https://www.movable-type.co.uk/scripts/latlong.html
double haversine(double lat1, double lon1, double lat2, double lon2) {
	const double φ1    = lat1 * πrad; // φ, λ in radians
	const double φ2    = lat2 * πrad;
	const double sinΔφ = sin((lat2 - lat1) * πrad / 2);
	const double sinΔλ = sin((lon2 - lon1) * πrad / 2);
	const double a     = sinΔφ*sinΔφ + cos(φ1)*cos(φ2)*sinΔλ*sinΔλ;
	const double c     = 2 * atan2(sqrt(a), sqrt(1-a));
	const double d     = Rearth * c; // in metres
	return d;
}
