angular.module("LibreDNSApp", ["ngRoute", "ngResource", "ngSanitize"])
    .config(function($routeProvider, $locationProvider) {
	$routeProvider
	    .when("/zones", {
		controller: "ZonesListController",
		templateUrl: "views/zone-list.html"
	    })
	    .when("/zones/:zoneId", {
		controller: "ZoneController",
		templateUrl: "views/zone.html"
	    })
	    .when("/", {
		templateUrl: "views/home.html"
	    });
	$locationProvider.html5Mode(true);
    });

String.prototype.capitalize = function() {
    return this
	.toLowerCase()
	.replace(
		/(^|\s|-)([a-z])/g,
	    function(m,p1,p2) { return p1+p2.toUpperCase(); }
	);
}

Array.prototype.inArray = function(v) {
    return this.reduce(function(presence, current) {
	return presence || current == v;
    }, false);
}

angular.module("LibreDNSApp")
    .directive('autofocus', ['$timeout', function($timeout) {
	return {
	    restrict: 'A',
	    link : function($scope, $element) {
		$timeout(function() {
		    $element[0].focus();
		});
	    }
	}
    }]);

angular.module("LibreDNSApp")
    .factory("Version", function($resource) {
	return $resource("/api/version")
    })
    .factory("Zone", function($resource) {
	return $resource("/api/zones/:zoneId", { zoneId: '@id' });
    })
    .factory("RR", function($resource) {
	return $resource("/api/zones/:zoneId/rr", { zoneId: '@id' }, {
	    remove: { method: "DELETE", hasBody: true, },
	});
    });

angular.module("LibreDNSApp")
    .filter("stripHTML", function() {
	return function(input) {
	    if (!input)
		return input;
	    return input.replace(
		/(<([^>]+)>)/ig,
		""
	    );
	}
    })
    .filter("capitalize", function() {
	return function(input) {
	    return input.capitalize();
	}
    })
    .directive('integer', function() {
	return {
	    require: 'ngModel',
	    link: function(scope, ele, attr, ctrl){
		ctrl.$parsers.unshift(function(viewValue){
		    return parseInt(viewValue, 10);
		});
	    }
	};
    })

    .directive('float', function() {
	return {
	    require: 'ngModel',
	    link: function(scope, ele, attr, ctrl){
		ctrl.$parsers.unshift(function(viewValue){
		    return parseFloat(viewValue, 10);
		});
	    }
	};
    })

    .run(function($rootScope, $http, $interval) {  })

    .controller("VersionController", function($scope, Version) {
	$scope.v = Version.get();
    })

    .controller("ZonesListController", function($scope, Zone, $location) {
	$scope.zones = Zone.query();

	$scope.show = function(id) {
	    $location.url("/zones/" + id);
	};
	$scope.attachZone = function() {
	    Zone.save({ domainName: $('#domainname').val() }, function() {
		$('#newZoneModal').modal('hide');
		$scope.zones = Zone.query();
	    }, function(response) {
		alert('An error occurs when trying to create zone: ' + response.data.errmsg);
	    });
	}
	$scope.deleteZone = function() {
	    Zone.delete({ zoneId: this.zone }, function() {
		$scope.zones = Zone.query();
	    }, function(response) {
		console.error('An error occurs when trying to delete zone:', response.data.errmsg);
	    });
	}
    })

    .controller("ZoneController", function($scope, Zone, $routeParams, $location) {
	$scope.zone = Zone.get({ zoneId: $routeParams.zoneId });

	$scope.saveZone = function() {
	    if (this.zone.id) {
		this.zone.$update();
	    } else {
		this.zone.$save(function() {
		    $location.url("/zones/" + $scope.zone.id);
		});
	    }
	}
	$scope.deleteZone = function() {
	    this.zone.$remove(function() {
		$location.url("/zones/");
	    }, function(response) {
		console.error('An error occurs when trying to delete zone:', response.data.errmsg);
	    });
	}
    })

    .controller("RRController", function($scope, RR, $routeParams, $location) {
	$scope.rrs = RR.query({ zoneId: $routeParams.zoneId });
	$scope.addRR = function() {
	    var rr = new RR();
	    rr.edit = true;
	    $scope.rrs.push(rr);
	}
	$scope.newRR = function(rr) {
	    rr.$save({ zoneId: $routeParams.zoneId }, function() {
		$scope.rrs = RR.query({ zoneId: $routeParams.zoneId });
	    }, function(response) {
		alert('An error occurs when trying to delete zone: ' + response.data.errmsg);
	    });
	}
	$scope.deleteRR = function(rr) {
	    rr.$remove({ zoneId: $routeParams.zoneId }, function() {
		$scope.rrs = RR.query({ zoneId: $routeParams.zoneId });
	    }, function(response) {
		console.error('An error occurs when trying to delete zone:', response.data.errmsg);
	    });
	}
    });
