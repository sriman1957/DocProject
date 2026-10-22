import 'package:docproject/core/network/api_client.dart';
import 'package:docproject/core/storage/token_storage.dart';
import 'package:docproject/features/groups/groups_controller.dart';
import 'package:docproject/features/groups/models/group.dart';
import 'package:docproject/features/groups/services/groups_service.dart';
import 'package:flutter_test/flutter_test.dart';

class FakeGroupsService extends GroupsService {
  FakeGroupsService({this.groupsResponse, this.error})
    : super(apiClient: ApiClient(), tokenStorage: TokenStorage());

  final List<Group>? groupsResponse;
  final Exception? error;
  int getGroupsCallCount = 0;

  @override
  Future<List<Group>> getGroups() async {
    getGroupsCallCount++;
    if (error != null) {
      throw error!;
    }
    return groupsResponse ?? const [];
  }
}

void main() {
  const sampleGroup1 = Group(
    id: 1,
    collegeId: 10,
    branchId: 2,
    name: 'CS 2024',
    description: 'First batch',
    createdBy: 5,
    isActive: true,
  );

  const sampleGroup2 = Group(
    id: 2,
    collegeId: 10,
    branchId: 3,
    name: 'EE 2024',
    description: 'Second batch',
    createdBy: 5,
    isActive: true,
  );

  group('GroupsController', () {
    test('initial state defaults to GroupsInitial', () {
      final service = FakeGroupsService();
      final controller = GroupsController(groupsService: service);

      expect(controller.state, isA<GroupsInitial>());
      expect(controller.groups, isEmpty);
      expect(service.getGroupsCallCount, 0);
    });

    test(
      'loadGroups transitions to GroupsLoaded on successful fetch',
      () async {
        final service = FakeGroupsService(
          groupsResponse: [sampleGroup1, sampleGroup2],
        );
        final controller = GroupsController(groupsService: service);

        final states = <GroupsState>[];
        controller.addListener(() => states.add(controller.state));

        await controller.loadGroups();

        expect(states, [isA<GroupsLoading>(), isA<GroupsLoaded>()]);

        expect(controller.state, isA<GroupsLoaded>());
        expect(controller.groups.length, 2);
        expect(controller.groups[0], sampleGroup1);
        expect(controller.groups[1], sampleGroup2);
        expect(service.getGroupsCallCount, 1);
      },
    );

    test(
      'loadGroups transitions to GroupsEmpty when no groups returned',
      () async {
        final service = FakeGroupsService(groupsResponse: []);
        final controller = GroupsController(groupsService: service);

        await controller.loadGroups();

        expect(controller.state, isA<GroupsEmpty>());
        expect(controller.groups, isEmpty);
        expect(service.getGroupsCallCount, 1);
      },
    );

    test('loadGroups transitions to GroupsError on ApiException', () async {
      final service = FakeGroupsService(
        error: const ApiException(
          statusCode: 500,
          message: 'internal server error',
        ),
      );
      final controller = GroupsController(groupsService: service);

      await controller.loadGroups();

      expect(controller.state, isA<GroupsError>());
      final errorState = controller.state as GroupsError;
      expect(errorState.message, 'internal server error');
      expect(errorState.statusCode, 500);
      expect(errorState.canRetry, isTrue);
    });

    test('loadGroups does not make repeated calls if already loaded and force is false', () async {
      final service = FakeGroupsService(groupsResponse: [sampleGroup1]);
      final controller = GroupsController(groupsService: service);

      await controller.loadGroups();
      expect(service.getGroupsCallCount, 1);

      // Re-trigger load without force
      await controller.loadGroups();
      expect(service.getGroupsCallCount, 1);
    });

    test('refresh forces reload of groups', () async {
      final service = FakeGroupsService(groupsResponse: [sampleGroup1]);
      final controller = GroupsController(groupsService: service);

      await controller.loadGroups();
      expect(service.getGroupsCallCount, 1);

      await controller.refresh();
      expect(service.getGroupsCallCount, 2);
    });

    test(
      'getGroupById finds group by ID or returns null if not found',
      () async {
        final service = FakeGroupsService(
          groupsResponse: [sampleGroup1, sampleGroup2],
        );
        final controller = GroupsController(groupsService: service);

        await controller.loadGroups();

        expect(controller.getGroupById(1), sampleGroup1);
        expect(controller.getGroupById(2), sampleGroup2);
        expect(controller.getGroupById(999), isNull);
      },
    );
  });
}
