import 'package:docproject/core/network/api_client.dart';
import 'package:docproject/core/storage/token_storage.dart';
import 'package:docproject/features/auth/models/user.dart';
import 'package:docproject/features/groups/groups_controller.dart';
import 'package:docproject/features/groups/models/group.dart';
import 'package:docproject/features/groups/screens/group_details_screen.dart';
import 'package:docproject/features/groups/screens/groups_listing_screen.dart';
import 'package:docproject/features/groups/services/groups_service.dart';
import 'package:flutter/material.dart';
import 'package:flutter_test/flutter_test.dart';

class MockGroupsService extends GroupsService {
  MockGroupsService({this.groupsResponse, this.error})
    : super(apiClient: ApiClient(), tokenStorage: TokenStorage());

  final List<Group>? groupsResponse;
  final Exception? error;

  @override
  Future<List<Group>> getGroups() async {
    if (error != null) throw error!;
    return groupsResponse ?? const [];
  }
}

void main() {
  const sampleStudent = User(
    id: 1,
    collegeId: 10,
    fullName: 'Student User',
    email: 'student@college.edu',
    role: 'student',
  );

  const sampleFaculty = User(
    id: 2,
    collegeId: 10,
    fullName: 'Prof. Miller',
    email: 'miller@college.edu',
    role: 'faculty',
    facultyCode: 'CS01',
  );

  const sampleGroup = Group(
    id: 42,
    collegeId: 10,
    branchId: 5,
    name: 'CS Batch 2024-2028',
    description: 'Computer Science primary batch',
    createdBy: 1,
    isActive: true,
    createdAt: '2026-09-01 10:00:00',
  );

  group('GroupsListingScreen', () {
    testWidgets('shows loading state when controller is in GroupsLoading', (
      tester,
    ) async {
      final controller = GroupsController(
        groupsService: MockGroupsService(),
        initialState: const GroupsLoading(),
      );

      await tester.pumpWidget(
        MaterialApp(
          home: Scaffold(
            body: GroupsListingScreen(
              user: sampleStudent,
              groupsController: controller,
              autoLoad: false,
            ),
          ),
        ),
      );

      expect(find.byType(CircularProgressIndicator), findsOneWidget);
      expect(find.text('Loading groups...'), findsOneWidget);
    });

    testWidgets('displays groups with cards for student role', (tester) async {
      final controller = GroupsController(
        groupsService: MockGroupsService(groupsResponse: [sampleGroup]),
        initialState: const GroupsLoaded([sampleGroup]),
      );

      await tester.pumpWidget(
        MaterialApp(
          home: Scaffold(
            body: GroupsListingScreen(
              user: sampleStudent,
              groupsController: controller,
              autoLoad: false,
            ),
          ),
        ),
      );

      expect(find.text('Enrolled Academic Groups'), findsOneWidget);
      expect(find.text('CS Batch 2024-2028'), findsOneWidget);
      expect(find.text('Active'), findsOneWidget);
      expect(find.text('Branch #5'), findsOneWidget);
      expect(find.text('Computer Science primary batch'), findsOneWidget);
    });

    testWidgets(
      'displays groups with faculty-oriented header for faculty role',
      (tester) async {
        final controller = GroupsController(
          groupsService: MockGroupsService(groupsResponse: [sampleGroup]),
          initialState: const GroupsLoaded([sampleGroup]),
        );

        await tester.pumpWidget(
          MaterialApp(
            home: Scaffold(
              body: GroupsListingScreen(
                user: sampleFaculty,
                groupsController: controller,
                autoLoad: false,
              ),
            ),
          ),
        );

        expect(find.text('Assigned Batch Groups'), findsOneWidget);
        expect(find.text('CS Batch 2024-2028'), findsOneWidget);
      },
    );

    testWidgets('shows empty state when no groups exist', (tester) async {
      final controller = GroupsController(
        groupsService: MockGroupsService(groupsResponse: []),
        initialState: const GroupsEmpty(),
      );

      await tester.pumpWidget(
        MaterialApp(
          home: Scaffold(
            body: GroupsListingScreen(
              user: sampleStudent,
              groupsController: controller,
              autoLoad: false,
            ),
          ),
        ),
      );

      expect(find.text('No Groups Found'), findsOneWidget);
      expect(
        find.textContaining('You are not enrolled in any academic groups yet'),
        findsOneWidget,
      );
      expect(find.text('Refresh'), findsOneWidget);
    });

    testWidgets('shows error state with retry action on failure', (
      tester,
    ) async {
      final controller = GroupsController(
        groupsService: MockGroupsService(),
        initialState: const GroupsError(
          message: 'Unable to connect to server',
          canRetry: true,
        ),
      );

      await tester.pumpWidget(
        MaterialApp(
          home: Scaffold(
            body: GroupsListingScreen(
              user: sampleStudent,
              groupsController: controller,
              autoLoad: false,
            ),
          ),
        ),
      );

      expect(find.text('Failed to Load Groups'), findsOneWidget);
      expect(find.text('Unable to connect to server'), findsOneWidget);
      expect(find.text('Try Again'), findsOneWidget);
    });

    testWidgets('tapping a group card navigates to GroupDetailsScreen', (
      tester,
    ) async {
      final controller = GroupsController(
        groupsService: MockGroupsService(groupsResponse: [sampleGroup]),
        initialState: const GroupsLoaded([sampleGroup]),
      );

      await tester.pumpWidget(
        MaterialApp(
          home: Scaffold(
            body: GroupsListingScreen(
              user: sampleStudent,
              groupsController: controller,
              autoLoad: false,
            ),
          ),
        ),
      );

      final groupCardFinder = find.text('CS Batch 2024-2028');
      expect(groupCardFinder, findsOneWidget);

      await tester.tap(groupCardFinder);
      await tester.pumpAndSettle();

      expect(find.byType(GroupDetailsScreen), findsOneWidget);
      expect(find.text('Group Details'), findsOneWidget);
      expect(find.text('#42'), findsWidgets); // ID row and banner
    });
  });
}
