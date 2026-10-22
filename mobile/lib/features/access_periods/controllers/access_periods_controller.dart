import 'package:flutter/foundation.dart';

import '../../../core/network/api_client.dart';
import '../models/access_period.dart';
import '../services/access_periods_service.dart';

sealed class AccessPeriodsState {
  const AccessPeriodsState();
}

class AccessPeriodsInitial extends AccessPeriodsState {
  const AccessPeriodsInitial();
}

class AccessPeriodsLoading extends AccessPeriodsState {
  const AccessPeriodsLoading();
}

class AccessPeriodsLoaded extends AccessPeriodsState {
  const AccessPeriodsLoaded(this.periods);

  final List<AccessPeriod> periods;

  AccessPeriod? get currentActivePeriod {
    final now = DateTime.now();
    for (final period in periods) {
      if (period.isActiveNow(now)) return period;
    }
    return null;
  }

  @override
  bool operator ==(Object other) =>
      identical(this, other) ||
      other is AccessPeriodsLoaded &&
          runtimeType == other.runtimeType &&
          listEquals(periods, other.periods);

  @override
  int get hashCode => periods.hashCode;
}

class AccessPeriodsEmpty extends AccessPeriodsState {
  const AccessPeriodsEmpty();
}

class AccessPeriodsError extends AccessPeriodsState {
  const AccessPeriodsError({
    required this.message,
    this.canRetry = true,
    this.statusCode,
  });

  final String message;
  final bool canRetry;
  final int? statusCode;

  @override
  bool operator ==(Object other) =>
      identical(this, other) ||
      other is AccessPeriodsError &&
          runtimeType == other.runtimeType &&
          message == other.message &&
          canRetry == other.canRetry &&
          statusCode == other.statusCode;

  @override
  int get hashCode =>
      message.hashCode ^ canRetry.hashCode ^ statusCode.hashCode;
}

class AccessPeriodsController extends ChangeNotifier {
  AccessPeriodsController({
    required this.accessPeriodsService,
    required this.groupId,
    required this.subgroupId,
    AccessPeriodsState initialState = const AccessPeriodsInitial(),
  }) : _state = initialState;

  final AccessPeriodsService accessPeriodsService;
  final int groupId;
  final int subgroupId;

  AccessPeriodsState _state;
  AccessPeriodsState get state => _state;

  bool _isCreating = false;
  bool get isCreating => _isCreating;

  Future<void> loadAccessPeriods({bool force = false}) async {
    if (!force &&
        (_state is AccessPeriodsLoading || _state is AccessPeriodsLoaded)) {
      return;
    }

    _state = const AccessPeriodsLoading();
    notifyListeners();

    try {
      final periods = await accessPeriodsService.getSubgroupAccessPeriods(
        groupId: groupId,
        subgroupId: subgroupId,
      );

      if (periods.isEmpty) {
        _state = const AccessPeriodsEmpty();
      } else {
        _state = AccessPeriodsLoaded(periods);
      }
    } on ApiException catch (e) {
      String message = e.message;
      if (e.statusCode == 403) {
        message = 'Access denied. Only assigned faculty members or admins can manage collection windows.';
      } else if (e.statusCode == 404) {
        message = 'Subgroup not found or inactive.';
      }
      _state = AccessPeriodsError(message: message, statusCode: e.statusCode);
    } catch (_) {
      _state = const AccessPeriodsError(
        message: 'Network error. Please check your connection and try again.',
      );
    } finally {
      notifyListeners();
    }
  }

  Future<void> refresh() => loadAccessPeriods(force: true);

  Future<AccessPeriod> createAccessPeriod({
    required DateTime startsAt,
    required DateTime endsAt,
  }) async {
    _isCreating = true;
    notifyListeners();

    try {
      final created = await accessPeriodsService.createAccessPeriod(
        groupId: groupId,
        subgroupId: subgroupId,
        startsAt: startsAt,
        endsAt: endsAt,
      );

      // Refresh list to incorporate newly created period
      await refresh();
      return created;
    } finally {
      _isCreating = false;
      notifyListeners();
    }
  }
}
