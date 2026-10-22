import 'package:docproject/core/network/api_client.dart';
import 'package:docproject/core/storage/token_storage.dart';
import 'package:docproject/features/auth/models/user.dart';
import 'package:docproject/features/groups/models/group.dart';
import 'package:docproject/features/subgroups/models/subgroup.dart';
import 'package:docproject/features/subgroups/screens/subgroup_details_screen.dart';
import 'package:docproject/features/subgroups/screens/subgroups_listing_screen.dart';
import 'package:docproject/features/subgroups/services/subgroups_service.dart';
import 'package:docproject/features/subgroups/subgroups_controller.dart';
import 'package:flutter/material.dart';
import 'package:flutter_test/flutter_test.dart';

class MockSubgroupsService extends SubgroupsService {
  MockSubgroupsService({this.subgroupsResponse, this.error})
    : super(apiClient: ApiClient(), tokenStorage: TokenStorage());

  final List<Subgroup>? subgroupsResponse;
  final Exception? error;

  @override
  Future<List<Subgroup>> getSubgroups(int groupId) async {
    if (error != null) throw error!;
    return subgroupsResponse ?? const [];
  }
}

void main() {
  const sampleStudent = User(
    id: 1,
    collegeId: 10,
    fullName: 'Test Student',
    email: 'student@college.edu',
    role: 'student',
  );

  const sampleParentGroup = Group(
    id: 42,
    collegeId: 10,
    branchId: 5,
    name: 'CS Batch 2024-2028',
    description: 'Primary department batch',
    createdBy: 1,
  );

  const sampleSubgroup = Subgroup(
    id: 99,
    collegeId: 10,
    groupId: 42,
    name: 'Midterm Submission',
    description: 'Upload semester certificates here',
    createdBy: 1,
    isActive: true,
  );

  group('SubgroupsListingScreen', () {
    testWidgets('shows loading state when controller is in SubgroupsLoading', (
      tester,
    ) async {
      final controller = SubgroupsController(
        subgroupsService: MockSubgroupsService(),
        initialState: const SubgroupsLoading(),
      );

      await tester.pumpWidget(
        MaterialApp(
          home: SubgroupsListingScreen(
            parentGroup: sampleParentGroup,
            user: sampleStudent,
            subgroupsController: controller,
            autoLoad: false,
          ),
        ),
      );

      expect(find.byType(CircularProgressIndicator), findsOneWidget);
      expect(find.text('Loading subgroups...'), findsOneWidget);
    });

    testWidgets('displays subgroups list with parent group context', (
      tester,
    ) async {
      final controller = SubgroupsController(
        subgroupsService: MockSubgroupsService(
          subgroupsResponse: [sampleSubgroup],
        ),
        initialState: const SubgroupsLoaded([sampleSubgroup], groupId: 42),
      );

      await tester.pumpWidget(
        MaterialApp(
          home: SubgroupsListingScreen(
            parentGroup: sampleParentGroup,
            user: sampleStudent,
            subgroupsController: controller,
            autoLoad: false,
          ),
        ),
      );

      expect(
        find.text('Parent Group: CS Batch 2024-2028 (#42)'),
        findsOneWidget,
      );
      expect(find.text('Collection Subgroups'), findsOneWidget);
      expect(find.text('Midterm Submission'), findsOneWidget);
      expect(find.text('Active'), findsOneWidget);
      expect(find.text('Subgroup #99'), findsOneWidget);
      expect(find.text('Upload semester certificates here'), findsOneWidget);
    });

    testWidgets('shows empty state when no subgroups exist for group', (
      tester,
    ) async {
      final controller = SubgroupsController(
        subgroupsService: MockSubgroupsService(subgroupsResponse: []),
        initialState: const SubgroupsEmpty(groupId: 42),
      );

      await tester.pumpWidget(
        MaterialApp(
          home: SubgroupsListingScreen(
            parentGroup: sampleParentGroup,
            user: sampleStudent,
            subgroupsController: controller,
            autoLoad: false,
          ),
        ),
      );

      expect(find.text('No Subgroups Available'), findsOneWidget);
      expect(
        find.textContaining(
          'No document collection activities have been created',
        ),
        findsOneWidget,
      );
      expect(find.text('Refresh'), findsOneWidget);
    });

    testWidgets('shows error state with retry action on failure', (
      tester,
    ) async {
      final controller = SubgroupsController(
        subgroupsService: MockSubgroupsService(),
        initialState: const SubgroupsError(
          message: 'group not found',
          canRetry: true,
          statusCode: 404,
        ),
      );

      await tester.pumpWidget(
        MaterialApp(
          home: SubgroupsListingScreen(
            parentGroup: sampleParentGroup,
            user: sampleStudent,
            subgroupsController: controller,
            autoLoad: false,
          ),
        ),
      );

      expect(find.text('Failed to Load Subgroups'), findsOneWidget);
      expect(find.text('group not found'), findsOneWidget);
      expect(find.text('Try Again'), findsOneWidget);
    });

    testWidgets('tapping a subgroup card opens SubgroupDetailsScreen', (
      tester,
    ) async {
      final controller = SubgroupsController(
        subgroupsService: MockSubgroupsService(
          subgroupsResponse: [sampleSubgroup],
        ),
        initialState: const SubgroupsLoaded([sampleSubgroup], groupId: 42),
      );

      await tester.pumpWidget(
        MaterialApp(
          home: SubgroupsListingScreen(
            parentGroup: sampleParentGroup,
            user: sampleStudent,
            subgroupsController: controller,
            autoLoad: false,
          ),
        ),
      );

      final cardFinder = find.text('Midterm Submission');
      expect(cardFinder, findsOneWidget);

      await tester.tap(cardFinder);
      await tester.pumpAndSettle();

      expect(find.byType(SubgroupDetailsScreen), findsOneWidget);
      expect(find.text('Subgroup Details'), findsOneWidget);
      expect(find.text('Active Subgroup'), findsOneWidget);
      expect(
        find.text('Parent Group: CS Batch 2024-2028 (#42)'),
        findsOneWidget,
      );
    });
  });
}
