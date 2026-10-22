import 'package:docproject/features/auth/models/user.dart';
import 'package:docproject/features/groups/models/group.dart';
import 'package:docproject/features/groups/screens/group_details_screen.dart';
import 'package:flutter/material.dart';
import 'package:flutter_test/flutter_test.dart';

void main() {
  const sampleStudent = User(
    id: 1,
    collegeId: 10,
    fullName: 'Jane Doe',
    email: 'jane@college.edu',
    role: 'student',
  );

  const sampleFaculty = User(
    id: 2,
    collegeId: 10,
    fullName: 'Prof. Davis',
    email: 'davis@college.edu',
    role: 'faculty',
    facultyCode: 'ENG02',
  );

  const sampleGroup = Group(
    id: 88,
    collegeId: 10,
    branchId: 4,
    name: 'Mechanical 2024',
    description: 'Mechanical Engineering primary section',
    createdBy: 15,
    isActive: true,
    createdAt: '2026-08-15 11:00:00',
    updatedAt: '2026-08-20 16:00:00',
  );

  group('GroupDetailsScreen', () {
    testWidgets('renders all group fields and metadata correctly', (
      tester,
    ) async {
      await tester.pumpWidget(
        const MaterialApp(
          home: GroupDetailsScreen(group: sampleGroup, user: sampleStudent),
        ),
      );

      expect(find.text('Group Details'), findsOneWidget);
      expect(find.text('Mechanical 2024'), findsOneWidget);
      expect(find.text('Active Batch'), findsOneWidget);
      expect(find.text('Group ID: #88 • Branch #4'), findsOneWidget);
      expect(
        find.text('Mechanical Engineering primary section'),
        findsOneWidget,
      );
      expect(find.text('#88'), findsOneWidget);
      expect(find.text('#4'), findsOneWidget);
      expect(find.text('#10'), findsOneWidget);
      expect(find.text('#15'), findsOneWidget);
      expect(find.text('2026-08-15 11:00:00'), findsOneWidget);
      expect(find.text('2026-08-20 16:00:00'), findsOneWidget);
    });

    testWidgets('shows student-tailored next phase notice for student', (
      tester,
    ) async {
      await tester.pumpWidget(
        const MaterialApp(
          home: GroupDetailsScreen(group: sampleGroup, user: sampleStudent),
        ),
      );

      expect(find.text('Document Collection Activities'), findsOneWidget);
      expect(
        find.textContaining('Subgroups and certificate submission requests'),
        findsOneWidget,
      );
    });

    testWidgets('shows faculty-tailored next phase notice for faculty', (
      tester,
    ) async {
      await tester.pumpWidget(
        const MaterialApp(
          home: GroupDetailsScreen(group: sampleGroup, user: sampleFaculty),
        ),
      );

      expect(find.text('Subgroups & Roster Management'), findsOneWidget);
      expect(
        find.textContaining(
          'Creating collection subgroups and managing student member enrollments',
        ),
        findsOneWidget,
      );
    });

    testWidgets('back button pops the screen and returns to previous view', (
      tester,
    ) async {
      await tester.pumpWidget(
        MaterialApp(
          home: Builder(
            builder: (context) => Scaffold(
              body: TextButton(
                onPressed: () {
                  Navigator.of(context).push(
                    MaterialPageRoute<void>(
                      builder: (_) => const GroupDetailsScreen(
                        group: sampleGroup,
                        user: sampleStudent,
                      ),
                    ),
                  );
                },
                child: const Text('Open Details'),
              ),
            ),
          ),
        ),
      );

      await tester.tap(find.text('Open Details'));
      await tester.pumpAndSettle();

      expect(find.byType(GroupDetailsScreen), findsOneWidget);

      final backButton = find.byType(BackButton);
      expect(backButton, findsOneWidget);

      await tester.tap(backButton);
      await tester.pumpAndSettle();

      expect(find.byType(GroupDetailsScreen), findsNothing);
      expect(find.text('Open Details'), findsOneWidget);
    });
  });
}
