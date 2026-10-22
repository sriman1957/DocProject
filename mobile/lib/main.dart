import 'package:flutter/material.dart';

import 'app.dart';
import 'app_dependencies.dart';

void main() {
  WidgetsFlutterBinding.ensureInitialized();
  final dependencies = AppDependencies.create();
  runApp(DocProjectApp(dependencies: dependencies));
}
