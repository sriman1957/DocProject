import 'package:docproject/core/network/api_client.dart';
import 'package:docproject/core/storage/token_storage.dart';
import 'package:docproject/features/access_periods/controllers/access_periods_controller.dart';
import 'package:docproject/features/access_periods/models/access_period.dart';
import 'package:docproject/features/access_periods/services/access_periods_service.dart';
import 'package:flutter_test/flutter_test.dart';

class FakeAccessPeriodsService extends AccessPeriodsService {
  FakeAccessPeriodsService()
    : super(
        apiClient: ApiClient(baseUrl: 'http://localhost'),
        tokenStorage: TokenStorage(),
      );

  List<AccessPeriod> periodsToReturn = const [];
  AccessPeriod? createdPeriodToReturn;
  Object? errorToThrow;
  int listCallCount = 0;
  int createCallCount = 0;

  @override
  Future<List<AccessPeriod>> getSubgroupAccessPeriods({
    required int groupId,
    required int subgroupId,
  }) async {
    listCallCount++;
    if (errorToThrow != null) throw errorToThrow!;
    return periodsToReturn;
  }

  @override
  Future<AccessPeriod> createAccessPeriod({
    required int groupId,
    required int subgroupId,
    required DateTime startsAt,
    required DateTime endsAt,
  }) async {
    createCallCount++;
    if (errorToThrow != null) throw errorToThrow!;
    return createdPeriodToReturn!;
  }
}

void main() {
  group('AccessPeriodsController', () {
    late FakeAccessPeriodsService fakeService;
    late AccessPeriodsController controller;

    final samplePeriod = AccessPeriod(
      id: 1,
      subgroupId: 10,
      groupId: 5,
      collegeId: 2,
      startsAt: DateTime.now().subtract(const Duration(days: 1)),
      endsAt: DateTime.now().add(const Duration(days: 5)),
      createdBy: 8,
    );

    setUp(() {
      fakeService = FakeAccessPeriodsService();
      controller = AccessPeriodsController(
        accessPeriodsService: fakeService,
        groupId: 5,
        subgroupId: 10,
      );
    });

    test('initial state is AccessPeriodsInitial', () {
      expect(controller.state, isA<AccessPeriodsInitial>());
      expect(controller.isCreating, isFalse);
    });

    test(
      'loadAccessPeriods transitions to Loading then Loaded when results exist',
      () async {
        fakeService.periodsToReturn = [samplePeriod];

        await controller.loadAccessPeriods();

        expect(controller.state, isA<AccessPeriodsLoaded>());
        final loaded = controller.state as AccessPeriodsLoaded;
        expect(loaded.periods.length, 1);
        expect(loaded.currentActivePeriod, isNotNull);
        expect(fakeService.listCallCount, 1);
      },
    );

    test(
      'loadAccessPeriods transitions to Empty when no periods exist',
      () async {
        fakeService.periodsToReturn = [];

        await controller.loadAccessPeriods();

        expect(controller.state, isA<AccessPeriodsEmpty>());
      },
    );

    test('loadAccessPeriods transitions to Error on 403 Forbidden', () async {
      fakeService.errorToThrow = const ApiException(
        statusCode: 403,
        message: 'Forbidden',
      );

      await controller.loadAccessPeriods();

      expect(controller.state, isA<AccessPeriodsError>());
      final err = controller.state as AccessPeriodsError;
      expect(err.statusCode, 403);
      expect(err.message, contains('Only assigned faculty members'));
    });

    test('createAccessPeriod creates period and reloads list', () async {
      fakeService.createdPeriodToReturn = samplePeriod;
      fakeService.periodsToReturn = [samplePeriod];

      final created = await controller.createAccessPeriod(
        startsAt: samplePeriod.startsAt,
        endsAt: samplePeriod.endsAt,
      );

      expect(created.id, samplePeriod.id);
      expect(fakeService.createCallCount, 1);
      // refresh() should have been called
      expect(fakeService.listCallCount, 1);
      expect(controller.state, isA<AccessPeriodsLoaded>());
    });
  });
}
