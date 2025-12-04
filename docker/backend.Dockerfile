FROM python:3.11-slim
WORKDIR /app
COPY src/backend/requirements.txt .
RUN pip install --no-cache-dir -r requirements.txt
COPY src/backend /app
ENV RUNS_DIR=/app/runs
RUN mkdir -p /app/runs
CMD ["uvicorn", "main:app", "--host", "0.0.0.0", "--port", "8000"]
