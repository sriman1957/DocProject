import 'package:docproject/app.dart';
import 'package:flutter_test/flutter_test.dart';

void main() {
  testWidgets('DocProject app launches successfully', (
    WidgetTester tester,
  ) async {
    await tester.pumpWidget(const DocProjectApp());

    expect(find.text('DocProject'), findsOneWidget);
    expect(find.text('Login screen foundation'), findsOneWidget);
  });
}
