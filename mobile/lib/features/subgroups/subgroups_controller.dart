import 'package:flutter/foundation.dart';

import '../../core/network/api_client.dart';
import 'models/subgroup.dart';
import 'services/subgroups_service.dart';

sealed class SubgroupsState {
  const SubgroupsState();
}

class SubgroupsInitial extends SubgroupsState {
  const SubgroupsInitial();
}

class SubgroupsLoading extends SubgroupsState {
  const SubgroupsLoading();
}

class SubgroupsLoaded extends SubgroupsState {
  const SubgroupsLoaded(this.subgroups, {this.groupId});

  final List<Subgroup> subgroups;
  final int? groupId;

  @override
  bool operator ==(Object other) =>
      identical(this, other) ||
      other is SubgroupsLoaded &&
          runtimeType == other.runtimeType &&
          groupId == other.groupId &&
          listEquals(subgroups, other.subgroups);

  @override
  int get hashCode => subgroups.hashCode ^ groupId.hashCode;
}

class SubgroupsEmpty extends SubgroupsState {
  const SubgroupsEmpty({this.groupId});

  final int? groupId;

  @override
  bool operator ==(Object other) =>
      identical(this, other) ||
      other is SubgroupsEmpty &&
          runtimeType == other.runtimeType &&
          groupId == other.groupId;

  @override
  int get hashCode => groupId.hashCode;
}

class SubgroupsError extends SubgroupsState {
  const SubgroupsError({
    required this.message,
    this.canRetry = true,
    this.statusCode,
    this.groupId,
  });

  final String message;
  final bool canRetry;
  final int? statusCode;
  final int? groupId;

  @override
  bool operator ==(Object other) =>
      identical(this, other) ||
      other is SubgroupsError &&
          runtimeType == other.runtimeType &&
          message == other.message &&
          canRetry == other.canRetry &&
          statusCode == other.statusCode &&
          groupId == other.groupId;

  @override
  int get hashCode =>
      message.hashCode ^
      canRetry.hashCode ^
      statusCode.hashCode ^
      groupId.hashCode;
}

class SubgroupsController extends ChangeNotifier {
  SubgroupsController({
    required this.subgroupsService,
    SubgroupsState initialState = const SubgroupsInitial(),
  }) : _state = initialState;

  final SubgroupsService subgroupsService;

  SubgroupsState _state;
  SubgroupsState get state => _state;

  List<Subgroup> get subgroups => switch (_state) {
    SubgroupsLoaded(:final subgroups) => subgroups,
    _ => const [],
  };

  bool _isLoading = false;
  int? _loadedGroupId;

  Future<void> loadSubgroups(int groupId, {bool force = false}) async {
    if (_isLoading) return;
    if (!force &&
        _loadedGroupId == groupId &&
        (_state is SubgroupsLoaded || _state is SubgroupsEmpty)) {
      return;
    }

    _isLoading = true;
    _state = const SubgroupsLoading();
    notifyListeners();

    try {
      final result = await subgroupsService.getSubgroups(groupId);
      _loadedGroupId = groupId;
      if (result.isEmpty) {
        _state = SubgroupsEmpty(groupId: groupId);
      } else {
        _state = SubgroupsLoaded(result, groupId: groupId);
      }
    } on ApiException catch (e) {
      _state = SubgroupsError(
        message: e.message,
        statusCode: e.statusCode,
        groupId: groupId,
      );
    } catch (_) {
      _state = SubgroupsError(
        message: 'Unable to load subgroups. Please check your connection and try again.',
        groupId: groupId,
      );
    } finally {
      _isLoading = false;
      notifyListeners();
    }
  }

  Future<void> refresh(int groupId) => loadSubgroups(groupId, force: true);

  Subgroup? getSubgroupById(int id) {
    for (final subgroup in subgroups) {
      if (subgroup.id == id) return subgroup;
    }
    return null;
  }
}
