import 'package:docproject/core/network/api_client.dart';
import 'package:docproject/core/storage/token_storage.dart';
import 'package:docproject/features/access_periods/controllers/access_periods_controller.dart';
import 'package:docproject/features/access_periods/models/access_period.dart';
import 'package:docproject/features/access_periods/screens/subgroup_access_periods_screen.dart';
import 'package:docproject/features/access_periods/services/access_periods_service.dart';
import 'package:docproject/features/access_periods/widgets/access_period_card.dart';
import 'package:docproject/features/access_periods/widgets/create_access_period_sheet.dart';
import 'package:docproject/features/auth/models/user.dart';
import 'package:docproject/features/groups/models/group.dart';
import 'package:docproject/features/subgroups/models/subgroup.dart';
import 'package:flutter/material.dart';
import 'package:flutter_test/flutter_test.dart';

class FakeAccessPeriodsService extends AccessPeriodsService {
  FakeAccessPeriodsService()
    : super(
        apiClient: ApiClient(baseUrl: 'http://localhost'),
        tokenStorage: TokenStorage(),
      );

  List<AccessPeriod> periodsToReturn = const [];
  Object? errorToThrow;
  Duration? delay;
  int listCallCount = 0;

  @override
  Future<List<AccessPeriod>> getSubgroupAccessPeriods({
    required int groupId,
    required int subgroupId,
  }) async {
    listCallCount++;
    if (delay != null) await Future<void>.delayed(delay!);
    if (errorToThrow != null) throw errorToThrow!;
    return periodsToReturn;
  }
}

void main() {
  const facultyUser = User(
    id: 1,
    collegeId: 10,
    fullName: 'Prof. Miller',
    email: 'miller@college.edu',
    role: 'faculty',
    facultyCode: 'ENG01',
  );

  const studentUser = User(
    id: 2,
    collegeId: 10,
    fullName: 'Bob Student',
    email: 'bob@college.edu',
    role: 'student',
  );

  const testGroup = Group(
    id: 42,
    collegeId: 10,
    branchId: 1,
    name: 'CS Year 3',
    description: 'Junior year CS students',
    createdBy: 1,
  );

  const testSubgroup = Subgroup(
    id: 105,
    collegeId: 10,
    groupId: 42,
    name: 'Lab Assignments',
    description: 'Weekly lab assignment uploads',
    createdBy: 1,
    isActive: true,
  );

  final samplePeriod = AccessPeriod(
    id: 88,
    subgroupId: 105,
    groupId: 42,
    collegeId: 10,
    startsAt: DateTime.now().subtract(const Duration(days: 2)),
    endsAt: DateTime.now().add(const Duration(days: 5)),
    createdBy: 1,
  );

  Widget createWidget({
    required User user,
    required AccessPeriodsController controller,
    bool autoLoad = true,
  }) {
    return MaterialApp(
      home: SubgroupAccessPeriodsScreen(
        user: user,
        parentGroup: testGroup,
        subgroup: testSubgroup,
        controller: controller,
        autoLoad: autoLoad,
      ),
    );
  }

  group('SubgroupAccessPeriodsScreen', () {
    late FakeAccessPeriodsService fakeService;
    late AccessPeriodsController controller;

    setUp(() {
      fakeService = FakeAccessPeriodsService();
      controller = AccessPeriodsController(
        accessPeriodsService: fakeService,
        groupId: 42,
        subgroupId: 105,
      );
    });

    testWidgets('shows loading indicator while loading', (tester) async {
      fakeService.delay = const Duration(milliseconds: 300);

      await tester.pumpWidget(
        createWidget(user: facultyUser, controller: controller, autoLoad: true),
      );

      await tester.pump();
      expect(
        find.byKey(const Key('access_periods_loading_indicator')),
        findsOneWidget,
      );

      await tester.pump(const Duration(milliseconds: 300));
      await tester.pumpAndSettle();
    });

    testWidgets('displays list of access periods with metadata', (
      tester,
    ) async {
      fakeService.periodsToReturn = [samplePeriod];

      await tester.pumpWidget(
        createWidget(user: facultyUser, controller: controller, autoLoad: true),
      );

      await tester.pumpAndSettle();

      expect(find.text('Parent Group: CS Year 3'), findsOneWidget);
      expect(
        find.text('Managing collection windows for subgroup #105'),
        findsOneWidget,
      );
      expect(find.byType(AccessPeriodCard), findsOneWidget);
      expect(find.text('Window #88'), findsOneWidget);
      expect(find.text('Active (Open)'), findsOneWidget);
    });

    testWidgets('displays empty state when no collection windows configured', (
      tester,
    ) async {
      fakeService.periodsToReturn = [];

      await tester.pumpWidget(
        createWidget(user: facultyUser, controller: controller, autoLoad: true),
      );

      await tester.pumpAndSettle();

      expect(
        find.byKey(const Key('access_periods_empty_state')),
        findsOneWidget,
      );
      expect(find.text('No Collection Windows Configured'), findsOneWidget);
    });

    testWidgets('displays error state and retry button works', (tester) async {
      fakeService.errorToThrow = const ApiException(
        statusCode: 500,
        message: 'Server database error',
      );

      await tester.pumpWidget(
        createWidget(user: facultyUser, controller: controller, autoLoad: true),
      );

      await tester.pumpAndSettle();

      expect(
        find.byKey(const Key('access_periods_error_state')),
        findsOneWidget,
      );
      expect(find.text('Server database error'), findsOneWidget);

      fakeService.errorToThrow = null;
      fakeService.periodsToReturn = [samplePeriod];

      await tester.tap(find.text('Try Again'));
      await tester.pump();
      await tester.pumpAndSettle();

      expect(find.byType(AccessPeriodCard), findsOneWidget);
    });

    testWidgets('tapping New Window opens CreateAccessPeriodSheet', (
      tester,
    ) async {
      fakeService.periodsToReturn = [samplePeriod];

      await tester.pumpWidget(
        createWidget(user: facultyUser, controller: controller, autoLoad: true),
      );

      await tester.pumpAndSettle();

      final addBtn = find.byKey(const Key('add_access_period_button'));
      expect(addBtn, findsOneWidget);

      await tester.tap(addBtn);
      await tester.pumpAndSettle();

      expect(find.byType(CreateAccessPeriodSheet), findsOneWidget);
      expect(find.text('New Collection Window'), findsOneWidget);
    });

    testWidgets('blocks student with Faculty Access Only screen', (
      tester,
    ) async {
      await tester.pumpWidget(
        createWidget(user: studentUser, controller: controller, autoLoad: true),
      );

      await tester.pumpAndSettle();

      expect(find.text('Faculty Access Only'), findsOneWidget);
      expect(fakeService.listCallCount, 0);
    });
  });
}
