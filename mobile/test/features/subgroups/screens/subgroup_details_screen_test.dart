import 'package:docproject/features/auth/models/user.dart';
import 'package:docproject/features/groups/models/group.dart';
import 'package:docproject/features/subgroups/models/subgroup.dart';
import 'package:docproject/features/subgroups/screens/subgroup_details_screen.dart';
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

  const sampleParentGroup = Group(
    id: 42,
    collegeId: 10,
    branchId: 3,
    name: 'IT Batch 2024',
    description: 'IT primary batch',
    createdBy: 1,
  );

  const sampleSubgroup = Subgroup(
    id: 77,
    collegeId: 10,
    groupId: 42,
    name: 'Hackathon Certificates',
    description: 'Upload hackathon winner or participation certificates',
    createdBy: 1,
    isActive: true,
    createdAt: '2026-09-10 14:00:00',
    updatedAt: '2026-09-15 18:00:00',
  );

  group('SubgroupDetailsScreen', () {
    testWidgets('renders all subgroup fields and metadata correctly', (
      tester,
    ) async {
      await tester.pumpWidget(
        const MaterialApp(
          home: SubgroupDetailsScreen(
            subgroup: sampleSubgroup,
            parentGroup: sampleParentGroup,
            user: sampleStudent,
          ),
        ),
      );

      expect(find.text('Subgroup Details'), findsOneWidget);
      expect(find.text('Hackathon Certificates'), findsOneWidget);
      expect(find.text('Active Subgroup'), findsOneWidget);
      expect(find.text('Parent Group: IT Batch 2024 (#42)'), findsOneWidget);
      expect(
        find.text('Upload hackathon winner or participation certificates'),
        findsOneWidget,
      );
      expect(find.text('#77'), findsOneWidget);
      expect(find.text('#42'), findsOneWidget);
      expect(find.text('#10'), findsOneWidget);
      expect(find.text('#1'), findsOneWidget);
      expect(find.text('2026-09-10 14:00:00'), findsOneWidget);
      expect(find.text('2026-09-15 18:00:00'), findsOneWidget);
    });

    testWidgets('shows student-tailored next phase notice for student', (
      tester,
    ) async {
      await tester.pumpWidget(
        const MaterialApp(
          home: SubgroupDetailsScreen(
            subgroup: sampleSubgroup,
            parentGroup: sampleParentGroup,
            user: sampleStudent,
          ),
        ),
      );

      expect(find.text('Document Submission Window'), findsOneWidget);
      expect(
        find.textContaining(
          'Uploading academic certificates into this subgroup',
        ),
        findsOneWidget,
      );
    });

    testWidgets('shows faculty-tailored next phase notice for faculty', (
      tester,
    ) async {
      await tester.pumpWidget(
        const MaterialApp(
          home: SubgroupDetailsScreen(
            subgroup: sampleSubgroup,
            parentGroup: sampleParentGroup,
            user: sampleFaculty,
          ),
        ),
      );

      expect(find.text('Collection & Access Management'), findsOneWidget);
      expect(
        find.textContaining(
          'Configuring access periods and reviewing submitted student documents',
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
                      builder: (_) => const SubgroupDetailsScreen(
                        subgroup: sampleSubgroup,
                        parentGroup: sampleParentGroup,
                        user: sampleStudent,
                      ),
                    ),
                  );
                },
                child: const Text('Open Subgroup Details'),
              ),
            ),
          ),
        ),
      );

      await tester.tap(find.text('Open Subgroup Details'));
      await tester.pumpAndSettle();

      expect(find.byType(SubgroupDetailsScreen), findsOneWidget);

      final backButton = find.byType(BackButton);
      expect(backButton, findsOneWidget);

      await tester.tap(backButton);
      await tester.pumpAndSettle();

      expect(find.byType(SubgroupDetailsScreen), findsNothing);
      expect(find.text('Open Subgroup Details'), findsOneWidget);
    });
  });
}
