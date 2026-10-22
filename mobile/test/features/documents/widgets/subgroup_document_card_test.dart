import 'package:docproject/features/documents/models/vault_document.dart';
import 'package:docproject/features/documents/widgets/subgroup_document_card.dart';
import 'package:flutter/material.dart';
import 'package:flutter_test/flutter_test.dart';

void main() {
  group('SubgroupDocumentCard Widget Tests', () {
    const sampleDoc = VaultDocument(
      id: 88,
      originalFilename: 'project_report.pdf',
      mimeType: 'application/pdf',
      fileSizeBytes: 1048576,
      sha256:
          'abcdef1234567890abcdef1234567890abcdef1234567890abcdef1234567890',
      uploadedAt: null,
    );

    testWidgets(
      'renders document filename, extension, size, and sha256 fingerprint',
      (tester) async {
        await tester.pumpWidget(
          MaterialApp(
            home: Scaffold(
              body: SubgroupDocumentCard(
                document: sampleDoc,
                onDownload: () {},
              ),
            ),
          ),
        );

        expect(find.text('project_report.pdf'), findsOneWidget);
        expect(find.text('PDF'), findsOneWidget);
        expect(find.text('1.0 MB'), findsOneWidget);
        expect(find.text('#88'), findsOneWidget);
        expect(find.text('SHA-256: abcdef1234567890...'), findsOneWidget);
        expect(find.text('Download'), findsOneWidget);
      },
    );

    testWidgets('renders fallback MIME text when sha256 is empty', (
      tester,
    ) async {
      const docWithoutHash = VaultDocument(
        id: 89,
        originalFilename: 'notes.txt',
        mimeType: 'text/plain',
        fileSizeBytes: 256,
        sha256: '',
        uploadedAt: null,
      );

      await tester.pumpWidget(
        MaterialApp(
          home: Scaffold(
            body: SubgroupDocumentCard(
              document: docWithoutHash,
              onDownload: () {},
            ),
          ),
        ),
      );

      expect(find.text('MIME: text/plain'), findsOneWidget);
    });

    testWidgets('tapping download button fires onDownload callback', (
      tester,
    ) async {
      var downloadFired = false;

      await tester.pumpWidget(
        MaterialApp(
          home: Scaffold(
            body: SubgroupDocumentCard(
              document: sampleDoc,
              onDownload: () {
                downloadFired = true;
              },
            ),
          ),
        ),
      );

      final buttonFinder = find.byKey(const Key('download_button_88'));
      expect(buttonFinder, findsOneWidget);

      await tester.tap(buttonFinder);
      await tester.pump();

      expect(downloadFired, isTrue);
    });

    testWidgets(
      'disables download button and shows progress indicator when isDownloading is true',
      (tester) async {
        var downloadFired = false;

        await tester.pumpWidget(
          MaterialApp(
            home: Scaffold(
              body: SubgroupDocumentCard(
                document: sampleDoc,
                isDownloading: true,
                onDownload: () {
                  downloadFired = true;
                },
              ),
            ),
          ),
        );

        expect(find.text('Downloading...'), findsOneWidget);
        expect(find.byType(CircularProgressIndicator), findsOneWidget);

        final button = tester.widget<FilledButton>(
          find.byKey(const Key('download_button_88')),
        );
        expect(button.onPressed, isNull);

        await tester.tap(find.byKey(const Key('download_button_88')));
        await tester.pump();

        expect(downloadFired, isFalse);
      },
    );

    testWidgets('tapping card triggers onTap callback if provided', (
      tester,
    ) async {
      var tapped = false;

      await tester.pumpWidget(
        MaterialApp(
          home: Scaffold(
            body: SubgroupDocumentCard(
              document: sampleDoc,
              onDownload: () {},
              onTap: () {
                tapped = true;
              },
            ),
          ),
        ),
      );

      await tester.tap(find.text('project_report.pdf'));
      await tester.pump();

      expect(tapped, isTrue);
    });
  });
}
