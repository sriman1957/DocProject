import 'package:docproject/core/network/api_client.dart';
import 'package:docproject/core/storage/token_storage.dart';
import 'package:docproject/features/subgroups/models/subgroup.dart';
import 'package:docproject/features/subgroups/services/subgroups_service.dart';
import 'package:docproject/features/subgroups/subgroups_controller.dart';
import 'package:flutter_test/flutter_test.dart';

class FakeSubgroupsService extends SubgroupsService {
  FakeSubgroupsService({this.subgroupsResponse, this.error})
    : super(apiClient: ApiClient(), tokenStorage: TokenStorage());

  final List<Subgroup>? subgroupsResponse;
  final Exception? error;
  int getSubgroupsCallCount = 0;
  int? lastRequestedGroupId;

  @override
  Future<List<Subgroup>> getSubgroups(int groupId) async {
    getSubgroupsCallCount++;
    lastRequestedGroupId = groupId;
    if (error != null) throw error!;
    return subgroupsResponse ?? const [];
  }
}

void main() {
  const sampleSub1 = Subgroup(
    id: 1,
    collegeId: 10,
    groupId: 42,
    name: 'Activity 1',
    description: 'First activity',
    createdBy: 5,
    isActive: true,
  );

  const sampleSub2 = Subgroup(
    id: 2,
    collegeId: 10,
    groupId: 42,
    name: 'Activity 2',
    description: 'Second activity',
    createdBy: 5,
    isActive: true,
  );

  group('SubgroupsController', () {
    test('initial state defaults to SubgroupsInitial', () {
      final service = FakeSubgroupsService();
      final controller = SubgroupsController(subgroupsService: service);

      expect(controller.state, isA<SubgroupsInitial>());
      expect(controller.subgroups, isEmpty);
      expect(service.getSubgroupsCallCount, 0);
    });

    test('loadSubgroups transitions to SubgroupsLoaded on success', () async {
      final service = FakeSubgroupsService(
        subgroupsResponse: [sampleSub1, sampleSub2],
      );
      final controller = SubgroupsController(subgroupsService: service);

      final states = <SubgroupsState>[];
      controller.addListener(() => states.add(controller.state));

      await controller.loadSubgroups(42);

      expect(states, [isA<SubgroupsLoading>(), isA<SubgroupsLoaded>()]);

      expect(controller.state, isA<SubgroupsLoaded>());
      expect(controller.subgroups.length, 2);
      expect(controller.subgroups[0], sampleSub1);
      expect(controller.subgroups[1], sampleSub2);
      expect(service.getSubgroupsCallCount, 1);
      expect(service.lastRequestedGroupId, 42);
    });

    test(
      'loadSubgroups transitions to SubgroupsEmpty when no subgroups returned',
      () async {
        final service = FakeSubgroupsService(subgroupsResponse: []);
        final controller = SubgroupsController(subgroupsService: service);

        await controller.loadSubgroups(42);

        expect(controller.state, isA<SubgroupsEmpty>());
        expect(controller.subgroups, isEmpty);
        expect(service.getSubgroupsCallCount, 1);
      },
    );

    test(
      'loadSubgroups transitions to SubgroupsError on ApiException',
      () async {
        final service = FakeSubgroupsService(
          error: const ApiException(statusCode: 403, message: 'forbidden'),
        );
        final controller = SubgroupsController(subgroupsService: service);

        await controller.loadSubgroups(42);

        expect(controller.state, isA<SubgroupsError>());
        final errorState = controller.state as SubgroupsError;
        expect(errorState.message, 'forbidden');
        expect(errorState.statusCode, 403);
        expect(errorState.canRetry, isTrue);
      },
    );

    test(
      'loadSubgroups ignores duplicate call for same group when already loaded',
      () async {
        final service = FakeSubgroupsService(subgroupsResponse: [sampleSub1]);
        final controller = SubgroupsController(subgroupsService: service);

        await controller.loadSubgroups(42);
        expect(service.getSubgroupsCallCount, 1);

        await controller.loadSubgroups(42);
        expect(service.getSubgroupsCallCount, 1);
      },
    );

    test(
      'loadSubgroups re-fetches when requesting a different group ID',
      () async {
        final service = FakeSubgroupsService(subgroupsResponse: [sampleSub1]);
        final controller = SubgroupsController(subgroupsService: service);

        await controller.loadSubgroups(42);
        expect(service.getSubgroupsCallCount, 1);

        await controller.loadSubgroups(99);
        expect(service.getSubgroupsCallCount, 2);
        expect(service.lastRequestedGroupId, 99);
      },
    );

    test('refresh forces reload of subgroups for given group', () async {
      final service = FakeSubgroupsService(subgroupsResponse: [sampleSub1]);
      final controller = SubgroupsController(subgroupsService: service);

      await controller.loadSubgroups(42);
      expect(service.getSubgroupsCallCount, 1);

      await controller.refresh(42);
      expect(service.getSubgroupsCallCount, 2);
    });

    test(
      'getSubgroupById retrieves matching subgroup or returns null',
      () async {
        final service = FakeSubgroupsService(
          subgroupsResponse: [sampleSub1, sampleSub2],
        );
        final controller = SubgroupsController(subgroupsService: service);

        await controller.loadSubgroups(42);

        expect(controller.getSubgroupById(1), sampleSub1);
        expect(controller.getSubgroupById(2), sampleSub2);
        expect(controller.getSubgroupById(999), isNull);
      },
    );
  });
}
