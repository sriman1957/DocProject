import 'package:flutter/foundation.dart';

import '../../core/network/api_client.dart';
import 'models/group.dart';
import 'services/groups_service.dart';

sealed class GroupsState {
  const GroupsState();
}

class GroupsInitial extends GroupsState {
  const GroupsInitial();
}

class GroupsLoading extends GroupsState {
  const GroupsLoading();
}

class GroupsLoaded extends GroupsState {
  const GroupsLoaded(this.groups);
  final List<Group> groups;

  @override
  bool operator ==(Object other) =>
      identical(this, other) ||
      other is GroupsLoaded &&
          runtimeType == other.runtimeType &&
          listEquals(groups, other.groups);

  @override
  int get hashCode => groups.hashCode;
}

class GroupsEmpty extends GroupsState {
  const GroupsEmpty();
}

class GroupsError extends GroupsState {
  const GroupsError({
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
      other is GroupsError &&
          runtimeType == other.runtimeType &&
          message == other.message &&
          canRetry == other.canRetry &&
          statusCode == other.statusCode;

  @override
  int get hashCode =>
      message.hashCode ^ canRetry.hashCode ^ statusCode.hashCode;
}

class GroupsController extends ChangeNotifier {
  GroupsController({
    required this.groupsService,
    GroupsState initialState = const GroupsInitial(),
  }) : _state = initialState;

  final GroupsService groupsService;

  GroupsState _state;
  GroupsState get state => _state;

  List<Group> get groups => switch (_state) {
    GroupsLoaded(:final groups) => groups,
    _ => const [],
  };

  bool _isLoading = false;

  Future<void> loadGroups({bool force = false}) async {
    if (_isLoading) return;
    if (!force && (_state is GroupsLoaded || _state is GroupsEmpty)) {
      return;
    }

    _isLoading = true;
    _state = const GroupsLoading();
    notifyListeners();

    try {
      final result = await groupsService.getGroups();
      if (result.isEmpty) {
        _state = const GroupsEmpty();
      } else {
        _state = GroupsLoaded(result);
      }
    } on ApiException catch (e) {
      _state = GroupsError(message: e.message, statusCode: e.statusCode);
    } catch (e) {
      _state = const GroupsError(
        message: 'Unable to load groups. Please check your connection and try again.',
      );
    } finally {
      _isLoading = false;
      notifyListeners();
    }
  }

  Future<void> refresh() => loadGroups(force: true);

  Group? getGroupById(int id) {
    for (final group in groups) {
      if (group.id == id) return group;
    }
    return null;
  }
}
